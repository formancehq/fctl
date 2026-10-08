package cloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// IdentityInfo contains only available IDs from verified root claims. It is a
// snapshot taken after any initial root refresh; requests reload the identity.
type IdentityInfo struct {
	Organizations []string
	Stacks        map[string][]string
}

// MembershipClient authenticates control-plane requests. Organization routes
// obtain separate organization grants; root credentials never reach a stack.
// A coordinator must cover reload, refresh and CAS saves for shared sessions.
func MembershipClient(ctx context.Context, base *http.Client, root *Session, out io.Writer, open func(context.Context, string) error, coordinator Coordinator) (*http.Client, string, IdentityInfo, error) {
	if root == nil || root.MembershipToken == nil {
		return nil, "", IdentityInfo{}, errors.New("missing cloud identity; log in again")
	}
	options, err := identityOptions(root.Options)
	if err != nil {
		return nil, "", IdentityInfo{}, err
	}
	provider, config, err := membership(ctx, base, options)
	if err != nil {
		return nil, "", IdentityInfo{}, err
	}
	config.Scopes = identityScopes()
	m := &targetManager{root: root, options: options, base: base, provider: provider, config: config, out: out, open: open, coordinator: coordinator}
	if coordinator == nil {
		m.coordinator = m.inMemory
	}
	issuer, err := url.Parse(options.Issuer)
	if err != nil {
		return nil, "", IdentityInfo{}, errors.New("invalid Membership issuer")
	}
	s := &membershipTransport{ctx: ctx, manager: m, issuer: issuer}
	_, info, err := s.token(ctx, "")
	if err != nil {
		return nil, "", IdentityInfo{}, err
	}
	client := cloneClient(base)
	client.Transport, client.CheckRedirect = s, noRedirect
	return client, options.Issuer, info, nil
}

type membershipTransport struct {
	mu      sync.Mutex
	ctx     context.Context
	manager *targetManager
	issuer  *url.URL
}

func (s *membershipTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	organization, err := s.route(req.URL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(req.Context())
	//nolint:contextcheck // Client lifetime must cancel independent request contexts.
	stop := context.AfterFunc(s.ctx, cancel)
	if err := s.ctx.Err(); err != nil {
		stop()
		cancel()
		return nil, err
	}
	token, _, err := s.token(ctx, organization)
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	transport := &oauth2.Transport{Source: oauth2.StaticTokenSource(token), Base: baseTransport(s.manager.base)}
	resp, err := transport.RoundTrip(req.Clone(ctx))
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cleanup: func() { stop(); cancel() }}
	return resp, nil
}

func (s *membershipTransport) route(uri *url.URL) (string, error) {
	if uri == nil || uri.User != nil || !sameOrigin(s.issuer.String(), uri.String()) || uri.RawPath != "" || uri.Opaque != "" {
		return "", errors.New("membership client refused an unsafe request URL")
	}
	prefix := strings.TrimRight(s.issuer.Path, "/")
	if !strings.HasPrefix(uri.Path, prefix+"/") {
		return "", errors.New("membership client refused a request outside its issuer path")
	}
	route := strings.TrimSuffix(strings.TrimPrefix(uri.Path, prefix), "/")
	if path.Clean(route) != route {
		return "", errors.New("invalid Membership route")
	}
	if rootMembershipRoute(route) {
		return "", nil
	}
	parts := strings.Split(route, "/")
	if len(parts) < 3 || parts[1] != "organizations" || !validIdentifier(parts[2]) {
		return "", errors.New("unsupported Membership route")
	}
	return parts[2], nil
}

func rootMembershipRoute(route string) bool {
	if slices.Contains([]string{"/me", "/organizations", "/me/invitations", "/_info", "/.well-known/scopes"}, route) {
		return true
	}
	parts := strings.Split(route, "/")
	return len(parts) == 5 && parts[1] == "me" && parts[2] == "invitations" && validIdentifier(parts[3]) && slices.Contains([]string{"accept", "reject", "decline"}, parts[4])
}

func (s *membershipTransport) token(ctx context.Context, organization string) (*oauth2.Token, IdentityInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.manager
	if err := ctx.Err(); err != nil {
		return nil, IdentityInfo{}, err
	}
	if m.pendingSave {
		return nil, IdentityInfo{}, errors.New("cloud identity save failed; log in again")
	}
	var info IdentityInfo
	token, err := m.coordinator(ctx, func(current *Session, save func(*Session) error) (*oauth2.Token, error) {
		if save == nil {
			return nil, errors.New("cloud coordinator requires a session save function")
		}
		root, claims, err := m.membershipIdentity(ctx, current, organization, save)
		if err != nil {
			return nil, err
		}
		info, err = verifiedIdentityInfo(m.options, claims)
		if err != nil {
			return nil, err
		}
		*m.root = *copyRoot(root)
		if organization == "" {
			token := cleanToken(root.MembershipToken)
			if err := boundToken(token, time.Now()); err != nil {
				return nil, err
			}
			return token, nil
		}
		return m.organizationToken(ctx, root, claims, organization, save)
	})
	return token, info, err
}

func (m *targetManager) membershipIdentity(ctx context.Context, current *Session, organization string, save func(*Session) error) (*Session, identityClaims, error) {
	root, claims, err := m.currentIdentity(ctx, current, save)
	if err != nil || organization == "" || slices.ContainsFunc(claims.Organizations, func(org organizationAccess) bool { return org.ID == organization }) {
		return root, claims, err
	}
	if m.resolvedRefresh && m.refreshClaimsNeeded {
		return root, claims, nil
	}
	// An explicit organization may have been created after login. Refresh once
	// under the coordinator, then keep consulting current userinfo when the
	// provider leaves the historical ID token unchanged.
	m.refreshClaimsNeeded = true
	if m.resolvedRefresh {
		return m.userInfoAndSave(ctx, root, save)
	}
	return m.refreshAndSave(ctx, root, save)
}

func verifiedIdentityInfo(options Options, claims identityClaims) (IdentityInfo, error) {
	info := IdentityInfo{Stacks: make(map[string][]string)}
	for _, org := range claims.Organizations {
		if !validIdentifier(org.ID) || slices.Contains(info.Organizations, org.ID) {
			return IdentityInfo{}, errors.New("identity contains invalid or duplicate organization IDs")
		}
		info.Organizations = append(info.Organizations, org.ID)
		info.Stacks[org.ID] = []string{}
	}
	targets, err := availableTargets(options, claims)
	if err != nil {
		return IdentityInfo{}, err
	}
	for _, target := range targets {
		info.Stacks[target.Options.Organization] = append(info.Stacks[target.Options.Organization], target.Options.Stack)
	}
	slices.Sort(info.Organizations)
	return info, nil
}

func organizationScopes(claims identityClaims, id string) ([]string, error) {
	for _, org := range claims.Organizations {
		if org.ID != id {
			continue
		}
		// Historical root claims omit scopes. Ask only the known supported set;
		// Membership consent still enforces the actual organization policy.
		if org.Scopes == nil {
			return slices.Clone(knownOrganizationScopes), nil
		}
		scopes := intersectOrganizationScopes(org.Scopes)
		if len(scopes) == 0 {
			return nil, errors.New("identity grants no organization scopes")
		}
		return scopes, nil
	}
	return nil, errors.New("identity does not authorize the requested organization")
}

func intersectOrganizationScopes(scopes []string) []string {
	var result []string
	for _, scope := range knownOrganizationScopes {
		if slices.Contains(scopes, scope) {
			result = append(result, scope)
		}
	}
	return result
}

func (m *targetManager) organizationToken(ctx context.Context, root *Session, claims identityClaims, org string, save func(*Session) error) (*oauth2.Token, error) {
	scopes, err := organizationScopes(claims, org)
	if err != nil {
		return nil, err
	}
	child, err := m.cachedOrganization(ctx, root, org, scopes)
	if err != nil {
		return nil, err
	}
	if child == nil {
		child, err = m.authorizeOrganization(ctx, root, org, scopes)
		if err != nil {
			return nil, err
		}
		if err := m.saveOrganization(root, org, child, save); err != nil {
			return nil, err
		}
	}
	if !usable(child.MembershipToken) {
		child, refreshErr := m.refreshOrganization(ctx, child, org, scopes)
		if child != nil {
			if err := m.saveOrganization(root, org, child, save); err != nil {
				return nil, err
			}
		}
		if refreshErr != nil {
			return nil, refreshErr
		}
		return cleanToken(child.MembershipToken), nil
	}
	return cleanToken(child.MembershipToken), nil
}

func (m *targetManager) cachedOrganization(ctx context.Context, root *Session, org string, scopes []string) (*Session, error) {
	child := copyRoot(root.Organizations[org])
	if child == nil {
		return nil, nil
	}
	matches, err := m.verifyOrganization(ctx, child, org, scopes, true)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, nil
	}
	if err := boundSessionTokens(child); err != nil {
		return nil, err
	}
	return child, nil
}

func (m *targetManager) saveOrganization(root *Session, org string, child *Session, save func(*Session) error) error {
	if root.Organizations == nil {
		root.Organizations = make(map[string]*Session)
	}
	root.Organizations[org] = copyRoot(child)
	return m.saveRoot(root, save)
}

func (m *targetManager) authorizeOrganization(ctx context.Context, root *Session, org string, scopes []string) (*Session, error) {
	config := *m.config
	config.Scopes = append([]string{"openid", "offline_access"}, scopes...)
	token, raw, err := deviceLogin(ctx, m.base, m.options, &config, m.out, m.open,
		[]oauth2.AuthCodeOption{oauth2.SetAuthURLParam("organization_id", org), oauth2.SetAuthURLParam("id_token_hint", root.IDToken)}, nil)
	if err != nil {
		return nil, err
	}
	child := &Session{Options: m.options, IDToken: raw, MembershipToken: cleanToken(token)}
	child.Options.Organization = org
	matches, err := m.verifyOrganization(ctx, child, org, scopes, false)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errors.New("organization grant permissions differ from root identity")
	}
	return child, nil
}

func (m *targetManager) verifyOrganization(ctx context.Context, child *Session, org string, scopes []string, stored bool) (bool, error) {
	return m.verifyOrganizationProof(ctx, child, org, scopes, stored, stored)
}

func (m *targetManager) verifyOrganizationProof(ctx context.Context, child *Session, org string, scopes []string, storedID, storedAccess bool) (bool, error) {
	expected := m.options
	expected.Organization = org
	if child.Options != expected || child.StackURL != "" || child.StackToken != nil || child.MembershipToken == nil || child.IDToken == "" {
		return false, errors.New("invalid organization session; log in again")
	}
	_, id, err := verifyIdentity(ctx, m.provider, m.options, child.IDToken, storedID)
	if err != nil {
		return false, err
	}
	access, err := m.provider.Verifier(&oidc.Config{ClientID: m.options.ClientID, SkipExpiryCheck: storedAccess}).Verify(ctx, child.MembershipToken.AccessToken)
	if err != nil {
		return false, safeError(ctx, "verify organization access", err)
	}
	var claims struct {
		Organization string `json:"organization_id"`
		Scope        string `json:"scope"`
	}
	if err := access.Claims(&claims); err != nil {
		return false, errors.New("invalid organization access claims")
	}
	if claims.Organization != org || access.Subject != id.Subject {
		return false, errors.New("organization access identity differs from requested organization")
	}
	_, rootID, err := verifyIdentity(ctx, m.provider, m.options, m.root.IDToken, true)
	if err != nil {
		return false, err
	}
	if access.Subject != rootID.Subject {
		return false, errors.New("organization access subject differs from root identity")
	}
	granted := strings.Fields(claims.Scope)
	filtered := intersectOrganizationScopes(granted)
	for _, scope := range granted {
		if strings.HasPrefix(scope, "organization:") && !slices.Contains(knownOrganizationScopes, scope) {
			return false, nil
		}
	}
	child.MembershipToken.Expiry = minExpiry(child.MembershipToken.Expiry, access.Expiry)
	return slices.Equal(filtered, scopes), nil
}

func (m *targetManager) refreshOrganization(ctx context.Context, child *Session, org string, scopes []string) (*Session, error) {
	if child.MembershipToken.RefreshToken == "" {
		return nil, errors.New("organization session expired; log in again")
	}
	old := cleanToken(child.MembershipToken)
	old.Expiry = time.Now().Add(-time.Hour)
	config := *m.config
	config.Scopes = nil
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, authClient(ctx, m.base, m.options.Issuer))
	fresh, err := config.TokenSource(authCtx, old).Token()
	if err != nil {
		return nil, safeError(ctx, "organization refresh", err)
	}
	child.MembershipToken = cleanToken(fresh)
	storedID := true
	if raw, ok := fresh.Extra("id_token").(string); ok && raw != "" {
		child.IDToken = raw
		storedID = false
	}
	if err := m.validateOrganizationRefresh(ctx, child, org, scopes, storedID); err != nil {
		child.IDToken = ""
		return child, err
	}
	return child, nil
}

func (m *targetManager) validateOrganizationRefresh(ctx context.Context, child *Session, org string, scopes []string, storedID bool) error {
	if err := boundToken(child.MembershipToken, time.Now()); err != nil {
		return err
	}
	matches, err := m.verifyOrganizationProof(ctx, child, org, scopes, storedID, false)
	if err != nil {
		return err
	}
	if !matches {
		return errors.New("refreshed organization permissions differ from root identity; log in again")
	}
	return nil
}
