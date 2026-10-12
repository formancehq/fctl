package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// LoginIdentity logs in without choosing an organization or stack. out is
// normally stderr. The optional browser callback is called once; its failure
// leaves the printed instructions available and does not prevent device login.
// Organization and Stack are ignored here; targeting is deferred to ClientForTarget.
func LoginIdentity(ctx context.Context, httpClient *http.Client, options Options, out io.Writer, open func(context.Context, string) error) (*Session, error) {
	options.Organization, options.Stack = "", ""
	options, err := identityOptions(options)
	if err != nil {
		return nil, err
	}
	provider, config, err := membership(ctx, httpClient, options)
	if err != nil {
		return nil, err
	}
	config.Scopes = identityScopes()
	token, raw, err := deviceLogin(ctx, httpClient, options, config, out, open, []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("prompt", "no-org")}, nil)
	if err != nil {
		return nil, err
	}
	claims, _, err := verifyIdentity(ctx, provider, options, raw, false)
	if err != nil {
		return nil, err
	}
	if _, err := availableTargets(options, claims); err != nil {
		return nil, err
	}
	return &Session{Options: options, IDToken: raw, MembershipToken: cleanToken(token), Targets: make(map[string]*Session)}, nil
}

func identityScopes() []string { return []string{"openid", "offline_access", "accesses", "on_behalf"} }

func identityOptions(options Options) (Options, error) {
	if options.Organization != "" || options.Stack != "" {
		return Options{}, errors.New("identity login does not select an organization or stack")
	}
	if options.Issuer == "" {
		options.Issuer = DefaultIssuer
	}
	options.Issuer = strings.TrimRight(options.Issuer, "/")
	if options.ClientID == "" {
		options.ClientID = "fctl"
	}
	if err := validateEndpoint(options.Issuer); err != nil {
		return Options{}, err
	}
	if !validIdentifier(options.ClientID) {
		return Options{}, errors.New("invalid cloud client ID")
	}
	return options, nil
}

func validIdentifier(value string) bool {
	return value != "" && !strings.ContainsAny(value, "/|?#\\") && !strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

func deviceLogin(ctx context.Context, base *http.Client, options Options, config *oauth2.Config, out io.Writer, open func(context.Context, string) error, authorization, poll []oauth2.AuthCodeOption) (*oauth2.Token, string, error) {
	if out == nil {
		return nil, "", errors.New("device authorization requires an output writer")
	}
	client := authClient(ctx, base, options.Issuer)
	client.Transport = &deviceTransport{base: client.Transport, endpoint: config.Endpoint.DeviceAuthURL, clientID: options.ClientID}
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, client)
	device, err := config.DeviceAuth(authCtx, authorization...)
	if err != nil {
		return nil, "", safeError(ctx, "device authorization", err)
	}
	if err := displayDevice(ctx, device, out, open); err != nil {
		return nil, "", err
	}
	// Membership already binds the authorized scopes to the device code.
	// Its token endpoint does not decode OAuth's space-delimited scope string.
	pollConfig := *config
	pollConfig.Scopes = nil
	token, err := pollConfig.DeviceAccessToken(authCtx, device, poll...)
	if err != nil {
		return nil, "", safeError(ctx, "device token", err)
	}
	if err := boundToken(token, time.Now()); err != nil {
		return nil, "", err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, "", errors.New("membership token response is missing an ID token")
	}
	return token, raw, nil
}

func displayDevice(ctx context.Context, device *oauth2.DeviceAuthResponse, out io.Writer, open func(context.Context, string) error) error {
	if device.DeviceCode == "" || device.UserCode == "" || device.Interval < 0 || device.Interval > 3600 || !device.Expiry.After(time.Now()) || device.Expiry.After(time.Now().Add(time.Hour)) {
		return errors.New("invalid device authorization response")
	}
	if len(device.UserCode) > 256 || strings.ContainsFunc(device.UserCode, unicode.IsControl) {
		return errors.New("invalid device user code")
	}
	uri, err := verificationURI(device.VerificationURI)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Open %s and enter code %s\n", device.VerificationURI, device.UserCode); err != nil {
		return errors.New("cannot display device authorization")
	}
	if open == nil {
		return ctx.Err()
	}
	query := uri.Query()
	query.Set("user_code", device.UserCode)
	uri.RawQuery = query.Encode()
	// Browser errors are deliberately not included in logs or the returned error.
	if err := open(ctx, uri.String()); err != nil {
		return ctx.Err()
	}
	return ctx.Err()
}

func verificationURI(raw string) (*url.URL, error) {
	uri, err := url.Parse(raw)
	if err != nil || strings.ContainsFunc(raw, unicode.IsControl) {
		return nil, errors.New("invalid device verification URI")
	}
	endpoint := *uri
	endpoint.RawQuery, endpoint.ForceQuery = "", false
	if err := validateEndpoint(endpoint.String()); err != nil {
		return nil, errors.New("invalid device verification URI")
	}
	if _, err := url.ParseQuery(uri.RawQuery); err != nil {
		return nil, errors.New("invalid device verification query")
	}
	return uri, nil
}

func verifyIdentity(ctx context.Context, provider *oidc.Provider, options Options, raw string, stored bool) (identityClaims, *oidc.IDToken, error) {
	token, err := provider.Verifier(&oidc.Config{ClientID: options.ClientID, SkipExpiryCheck: stored}).Verify(ctx, raw)
	if err != nil {
		return identityClaims{}, nil, safeError(ctx, "verify Membership identity", err)
	}
	var claims identityClaims
	if err := token.Claims(&claims); err != nil {
		return identityClaims{}, nil, errors.New("invalid Membership organization claims")
	}
	return claims, token, nil
}

type targetAccess struct {
	Options Options
	URI     string
	Scopes  []string
	Subject string
}

func availableTargets(options Options, claims identityClaims) ([]targetAccess, error) {
	var targets []targetAccess
	seen := make(map[string]bool)
	for _, organization := range claims.Organizations {
		if !validIdentifier(organization.ID) {
			return nil, errors.New("identity contains an invalid organization ID")
		}
		for _, stack := range organization.Stacks {
			target, err := availableTarget(options, organization.ID, stack)
			if err != nil {
				return nil, err
			}
			if target == nil {
				continue
			}
			key := targetKey(target.Options)
			if seen[key] {
				return nil, errors.New("identity contains a duplicate stack target")
			}
			seen[key] = true
			targets = append(targets, *target)
		}
	}
	slices.SortFunc(targets, func(a, b targetAccess) int { return strings.Compare(targetKey(a.Options), targetKey(b.Options)) })
	return targets, nil
}

func availableTarget(options Options, organization string, stack stackAccess) (*targetAccess, error) {
	if !validIdentifier(stack.ID) {
		return nil, errors.New("identity contains an invalid stack ID")
	}
	var scopes []string
	for _, scope := range []string{"stack:Read", "stack:Write"} {
		if slices.Contains(stack.Scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	if len(scopes) == 0 {
		return nil, nil
	}
	uri := strings.TrimRight(stack.URI, "/")
	if err := validateEndpoint(uri); err != nil {
		return nil, errors.New("identity contains an invalid stack URL")
	}
	options.Organization, options.Stack = organization, stack.ID
	return &targetAccess{Options: options, URI: uri, Scopes: scopes}, nil
}

func resolveTarget(options Options, claims identityClaims, desired Options) (targetAccess, error) {
	if desired.Issuer != "" && strings.TrimRight(desired.Issuer, "/") != options.Issuer {
		return targetAccess{}, errors.New("target issuer differs from cloud identity")
	}
	if desired.ClientID != "" && desired.ClientID != options.ClientID {
		return targetAccess{}, errors.New("target client ID differs from cloud identity")
	}
	targets, err := availableTargets(options, claims)
	if err != nil {
		return targetAccess{}, err
	}
	var matches []targetAccess
	for _, target := range targets {
		if desired.Organization != "" && desired.Organization != target.Options.Organization {
			continue
		}
		if desired.Stack != "" && desired.Stack != target.Options.Stack {
			continue
		}
		matches = append(matches, target)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(targets) == 0 {
		return targetAccess{}, errors.New("cloud identity grants no available stack access")
	}
	if len(matches) == 0 {
		return targetAccess{}, errors.New("selected Cloud stack is unavailable; check --organization and --stack\nList stacks: fctl cloud stack list --organization ORGANIZATION_ID")
	}
	return targetAccess{}, errors.New("choose a Cloud stack with --organization ORGANIZATION_ID --stack STACK_ID\nList organizations: fctl cloud organizations list\nList stacks: fctl cloud stack list --organization ORGANIZATION_ID")
}

func targetKey(options Options) string { return options.Organization + "/" + options.Stack }

// ClientForTarget resolves one stack from verified root identity claims, then
// obtains a stack-scoped Membership session. The root token is never exchanged
// at the stack. A coordinator reloads/saves the whole root and holds its auth
// lock across both root and scoped refreshes. nil is for a single in-memory
// caller; shared persisted sessions require the parent's coordinator.
func ClientForTarget(ctx context.Context, httpClient *http.Client, rootSession *Session, desired Options, out io.Writer, open func(context.Context, string) error, coordinator Coordinator) (*http.Client, string, error) {
	client, endpoint, _, err := ClientForResolvedTarget(ctx, httpClient, rootSession, desired, out, open, coordinator)
	return client, endpoint, err
}

// ClientForResolvedTarget also returns the verified target identity. Callers
// must retain this identity when automatic resolution selected a sole grant.
func ClientForResolvedTarget(ctx context.Context, httpClient *http.Client, rootSession *Session, desired Options, out io.Writer, open func(context.Context, string) error, coordinator Coordinator) (*http.Client, string, Options, error) {
	if rootSession == nil || rootSession.MembershipToken == nil {
		return nil, "", Options{}, errors.New("missing cloud identity; log in again")
	}
	options, err := identityOptions(rootSession.Options)
	if err != nil {
		return nil, "", Options{}, err
	}
	provider, config, err := membership(ctx, httpClient, options)
	if err != nil {
		return nil, "", Options{}, err
	}
	config.Scopes = identityScopes()
	manager := &targetManager{root: rootSession, options: options, desired: desired, base: httpClient, provider: provider, config: config, out: out, open: open, coordinator: coordinator}
	if coordinator == nil {
		manager.coordinator = manager.inMemory
	}
	if _, err := manager.coordinator(ctx, func(current *Session, save func(*Session) error) (*oauth2.Token, error) {
		child, err := manager.prepare(ctx, current, save)
		if err != nil {
			return nil, err
		}
		manager.child = child
		return cleanToken(child.MembershipToken), nil
	}); err != nil {
		return nil, "", Options{}, err
	}
	client, endpoint, err := Client(ctx, httpClient, manager.child, nil, manager.coordinateChild)
	if err != nil {
		return nil, "", Options{}, err
	}
	if manager.selected == nil {
		return nil, "", Options{}, errors.New("cloud stack resolution did not return a target")
	}
	return client, endpoint, manager.selected.Options, nil
}

type targetManager struct {
	mu                  sync.Mutex
	root                *Session
	options, desired    Options
	selected            *targetAccess
	child               *Session
	base                *http.Client
	provider            *oidc.Provider
	config              *oauth2.Config
	out                 io.Writer
	open                func(context.Context, string) error
	coordinator         Coordinator
	pendingSave         bool
	resolvedRefresh     bool
	refreshClaimsNeeded bool
}

func (m *targetManager) inMemory(ctx context.Context, work func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return work(copyRoot(m.root), func(next *Session) error { *m.root = *copyRoot(next); return nil })
}

func (m *targetManager) prepare(ctx context.Context, current *Session, save func(*Session) error) (*Session, error) {
	if m.pendingSave {
		return nil, errors.New("cloud identity save failed; log in again")
	}
	if save == nil {
		return nil, errors.New("cloud coordinator requires a session save function")
	}
	root, claims, err := m.currentIdentity(ctx, current, save)
	if err != nil {
		return nil, err
	}
	root, target, err := m.resolveCurrentTarget(ctx, root, claims, save)
	if err != nil {
		return nil, err
	}
	if m.selected != nil && (target.Options != m.selected.Options || target.URI != m.selected.URI) {
		return nil, errors.New("cloud target changed; recreate the client")
	}
	_, rootID, err := verifyIdentity(ctx, m.provider, m.options, root.IDToken, true)
	if err != nil {
		return nil, err
	}
	target.Subject = rootID.Subject
	m.selected = &target
	child, err := m.cachedOrAuthorize(ctx, root, target, save)
	if err != nil {
		return nil, err
	}
	guarded := copyRoot(child)
	guarded.allowedScopes, guarded.rootSubject = slices.Clone(target.Scopes), target.Subject
	state, err := validatedSession(ctx, m.provider, target.Options, guarded)
	if err != nil {
		return nil, err
	}
	if state.StackURL != target.URI {
		return nil, errors.New("cached target URL differs from root identity")
	}
	state.allowedScopes = slices.Clone(target.Scopes)
	*m.root = *copyRoot(root)
	return &state, nil
}

func (m *targetManager) resolveCurrentTarget(ctx context.Context, root *Session, claims identityClaims, save func(*Session) error) (*Session, targetAccess, error) {
	if _, err := availableTargets(m.options, claims); err != nil {
		return root, targetAccess{}, err
	}
	target, err := resolveTarget(m.options, claims, m.desired)
	if err == nil || m.resolvedRefresh || !refreshMissingTarget(m.options, claims, m.desired) {
		return root, target, err
	}
	// Newly created stacks may be absent from the login snapshot. Refresh once
	// while holding the coordinator, without starting another root device flow.
	m.resolvedRefresh = true
	m.refreshClaimsNeeded = true
	root, claims, err = m.refreshAndSave(ctx, root, save)
	if err != nil {
		return nil, targetAccess{}, err
	}
	target, err = resolveTarget(m.options, claims, m.desired)
	return root, target, err
}

func refreshMissingTarget(options Options, claims identityClaims, desired Options) bool {
	if !validIdentifier(desired.Stack) || (desired.Organization != "" && !validIdentifier(desired.Organization)) {
		return false
	}
	if desired.Issuer != "" && strings.TrimRight(desired.Issuer, "/") != options.Issuer {
		return false
	}
	if desired.ClientID != "" && desired.ClientID != options.ClientID {
		return false
	}
	return missingExplicitStack(claims, desired)
}

func missingExplicitStack(claims identityClaims, desired Options) bool {
	for _, org := range claims.Organizations {
		// A known stack paired with the wrong organization is an invalid target,
		// not a newly created stack missing from the signed login snapshot.
		if slices.ContainsFunc(org.Stacks, func(stack stackAccess) bool { return stack.ID == desired.Stack }) {
			return false
		}
	}
	return true
}

func (m *targetManager) cachedOrAuthorize(ctx context.Context, root *Session, target targetAccess, save func(*Session) error) (*Session, error) {
	child := root.Targets[targetKey(target.Options)]
	matches, err := m.matchesTarget(ctx, child, target)
	if err != nil {
		return nil, err
	}
	if !matches {
		var err error
		child, err = m.authorize(ctx, root, target)
		if err != nil {
			return nil, err
		}
		if root.Targets == nil {
			root.Targets = make(map[string]*Session)
		}
		root.Targets[targetKey(target.Options)] = child
		if err := m.saveRoot(root, save); err != nil {
			return nil, err
		}
	}
	return child, nil
}

func (m *targetManager) matchesTarget(ctx context.Context, child *Session, target targetAccess) (bool, error) {
	if child == nil {
		return false, nil
	}
	return verifyTargetSession(ctx, m.provider, target, child, true, true)
}

func (m *targetManager) coordinateChild(ctx context.Context, work func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
	return m.coordinator(ctx, func(current *Session, save func(*Session) error) (*oauth2.Token, error) {
		child, err := m.prepare(ctx, current, save)
		if err != nil {
			return nil, err
		}
		root := copyRoot(m.root)
		return work(child, func(updated *Session) error {
			root.Targets[targetKey(updated.Options)] = copyRoot(updated)
			return m.saveRoot(root, save)
		})
	})
}

func (m *targetManager) saveRoot(root *Session, save func(*Session) error) error {
	m.pendingSave = true
	*m.root = *copyRoot(root)
	if err := save(copyRoot(root)); err != nil {
		return errors.New("cannot save cloud identity")
	}
	m.pendingSave = false
	return nil
}

func (m *targetManager) currentIdentity(ctx context.Context, current *Session, save func(*Session) error) (*Session, identityClaims, error) {
	if current == nil || current.MembershipToken == nil {
		return nil, identityClaims{}, errors.New("missing cloud identity; log in again")
	}
	if current.IDToken == "" {
		return nil, identityClaims{}, errors.New("cloud identity is not verified; log in again")
	}
	options, err := identityOptions(current.Options)
	if err != nil || options != m.options {
		return nil, identityClaims{}, errors.New("cloud identity options changed; recreate the client")
	}
	claims, id, err := verifyIdentity(ctx, m.provider, options, current.IDToken, true)
	if err != nil {
		return nil, identityClaims{}, err
	}
	root := copyRoot(current)
	root.Options = options
	if !usable(root.MembershipToken) {
		return m.refreshAndSave(ctx, root, save)
	}
	if root.RequireUserInfo || m.refreshClaimsNeeded || !id.Expiry.After(time.Now().Add(refreshMargin)) {
		return m.userInfoAndSave(ctx, root, save)
	}
	return root, claims, nil
}

// Persist consumed refresh credentials even if identity verification fails.
// Save errors take precedence because retrying against an old revision is unsafe.
func (m *targetManager) refreshAndSave(ctx context.Context, root *Session, save func(*Session) error) (*Session, identityClaims, error) {
	m.resolvedRefresh = true
	renewed, claims, refreshErr := m.refreshRoot(ctx, root)
	if renewed != nil {
		if err := m.saveRoot(renewed, save); err != nil {
			return nil, identityClaims{}, err
		}
	}
	if refreshErr != nil {
		return nil, identityClaims{}, refreshErr
	}
	return renewed, claims, nil
}

func (m *targetManager) refreshRoot(ctx context.Context, root *Session) (*Session, identityClaims, error) {
	if root.MembershipToken.RefreshToken == "" {
		return nil, identityClaims{}, errors.New("cloud identity expired; log in again")
	}
	old := cleanToken(root.MembershipToken)
	old.Expiry = time.Now().Add(-time.Hour)
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, authClient(ctx, m.base, m.options.Issuer))
	fresh, err := m.config.TokenSource(authCtx, old).Token()
	if err != nil {
		return nil, identityClaims{}, safeError(ctx, "identity refresh", err)
	}
	root.MembershipToken = cleanToken(fresh)
	if err := boundToken(fresh, time.Now()); err != nil {
		root.MembershipToken.Expiry = time.Time{}
		root.IDToken = ""
		return root, identityClaims{}, err
	}
	root.MembershipToken = cleanToken(fresh)
	raw, ok := fresh.Extra("id_token").(string)
	if !ok || raw == "" {
		claims, _, err := verifyIdentity(ctx, m.provider, m.options, root.IDToken, false)
		if err != nil || root.RequireUserInfo || m.refreshClaimsNeeded {
			root.RequireUserInfo = true
			claims, err = m.currentUserInfo(ctx, root)
			return root, claims, err
		}
		return root, claims, nil
	}
	claims, _, err := verifyIdentity(ctx, m.provider, m.options, raw, false)
	if err != nil {
		// Retain rotated credentials, but invalidate the persisted identity proof.
		// A subsequent client must not reuse claims from before this failed refresh.
		root.IDToken = ""
		return root, identityClaims{}, err
	}
	root.IDToken, root.MembershipToken = raw, cleanToken(fresh)
	root.RequireUserInfo = false
	m.refreshClaimsNeeded = false
	return root, claims, nil
}

func (m *targetManager) authorize(ctx context.Context, root *Session, target targetAccess) (*Session, error) {
	config := *m.config
	config.Scopes = []string{"openid", "offline_access", "accesses"}
	resource := "stack://" + target.Options.Organization + "/" + target.Options.Stack + "|" + strings.Join(target.Scopes, " ")
	authorization := []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("organization_id", target.Options.Organization), oauth2.SetAuthURLParam("resource", resource), oauth2.SetAuthURLParam("id_token_hint", root.IDToken)}
	token, raw, err := deviceLogin(ctx, m.base, target.Options, &config, m.out, m.open, authorization, []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("resource", resource)})
	if err != nil {
		return nil, err
	}
	child := &Session{Options: target.Options, IDToken: raw, MembershipToken: cleanToken(token), StackURL: target.URI}
	matches, err := verifyTargetSession(ctx, m.provider, target, child, false, false)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errors.New("scoped session permissions differ from root identity")
	}
	return child, nil
}

func copyRoot(session *Session) *Session {
	if session == nil {
		return nil
	}
	cloned := *session
	cloned.allowedScopes = slices.Clone(session.allowedScopes)
	cloned.MembershipToken, cloned.StackToken = cleanToken(session.MembershipToken), cleanToken(session.StackToken)
	cloned.Targets = maps.Clone(session.Targets)
	cloned.Organizations = maps.Clone(session.Organizations)
	cloned.Applications = maps.Clone(session.Applications)
	cloneChildren(cloned.Targets)
	cloneChildren(cloned.Organizations)
	cloneChildren(cloned.Applications)
	return &cloned
}

func cloneChildren(children map[string]*Session) {
	for key, child := range children {
		if child == nil {
			continue
		}
		value := *child
		value.allowedScopes = slices.Clone(child.allowedScopes)
		value.MembershipToken, value.StackToken = cleanToken(child.MembershipToken), cleanToken(child.StackToken)
		value.Targets = nil
		value.Organizations = nil
		value.Applications = nil
		children[key] = &value
	}
}
