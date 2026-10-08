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

// ApplicationClient obtains a legacy application grant from Membership. The
// backend URL comes exclusively from the signed access-token audiences. Shared
// sessions require a coordinator holding the reload/refresh/save lock.
func ApplicationClient(ctx context.Context, base *http.Client, root *Session, organization, alias string, out io.Writer, open func(context.Context, string) error, coordinator Coordinator) (*http.Client, string, error) {
	if root == nil || root.MembershipToken == nil {
		return nil, "", errors.New("missing cloud identity; log in again")
	}
	if !validIdentifier(organization) || !validIdentifier(alias) {
		return nil, "", errors.New("application requires valid organization and alias")
	}
	options, err := identityOptions(root.Options)
	if err != nil {
		return nil, "", err
	}
	provider, config, err := membership(ctx, base, options)
	if err != nil {
		return nil, "", err
	}
	config.Scopes = identityScopes()
	m := &targetManager{root: root, options: options, base: base, provider: provider, config: config, out: out, open: open, coordinator: coordinator}
	if coordinator == nil {
		m.coordinator = m.inMemory
	}
	s := &applicationTransport{ctx: ctx, manager: m, organization: organization, alias: alias}
	if _, err := s.token(ctx, ""); err != nil {
		return nil, "", err
	}
	client := cloneClient(base)
	client.Transport, client.CheckRedirect = s, noRedirect
	return client, s.endpoint, nil
}

type applicationTransport struct {
	mu                            sync.Mutex
	ctx                           context.Context
	manager                       *targetManager
	organization, alias, endpoint string
}

func applicationGrant(claims identityClaims, organization, alias string) (applicationAccess, error) {
	var found *applicationAccess
	count := 0
	for _, org := range claims.Organizations {
		if org.ID != organization {
			continue
		}
		count++
		for _, app := range org.Applications {
			if app.Alias != alias {
				continue
			}
			if found != nil || !validIdentifier(app.ID) {
				return applicationAccess{}, errors.New("invalid or duplicate application grant")
			}
			value := app
			found = &value
		}
	}
	if count != 1 || found == nil {
		return applicationAccess{}, errors.New("identity does not authorize the requested application")
	}
	scopes := filterApplicationScopes(found.Scopes)
	if len(scopes) == 0 {
		return applicationAccess{}, errors.New("identity grants no supported application scopes")
	}
	found.Scopes = scopes
	return *found, nil
}

func filterApplicationScopes(granted []string) []string {
	var result []string
	for _, scope := range []string{"apps:Read", "apps:Write"} {
		if slices.Contains(granted, scope) {
			result = append(result, scope)
		}
	}
	return result
}

func applicationResource(alias string, scopes []string) string {
	return "app://" + alias + "|" + strings.Join(scopes, " ")
}

func (s *applicationTransport) token(ctx context.Context, method string) (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m := s.manager
	if m.pendingSave {
		return nil, errors.New("cloud identity save failed; log in again")
	}
	return m.coordinator(ctx, func(current *Session, save func(*Session) error) (*oauth2.Token, error) {
		if save == nil {
			return nil, errors.New("cloud coordinator requires a session save function")
		}
		root, claims, err := m.currentIdentity(ctx, current, save)
		if err != nil {
			return nil, err
		}
		grant, err := applicationGrant(claims, s.organization, s.alias)
		if err != nil {
			return nil, err
		}
		required := "apps:Read"
		if method != "" && !slices.Contains([]string{http.MethodGet, http.MethodHead, http.MethodOptions}, method) {
			required = "apps:Write"
		}
		if method != "" && !slices.Contains(grant.Scopes, required) {
			return nil, errors.New("identity does not grant the required application scope")
		}
		*m.root = *copyRoot(root)
		return s.applicationToken(ctx, root, grant, save)
	})
}

func (s *applicationTransport) applicationToken(ctx context.Context, root *Session, grant applicationAccess, save func(*Session) error) (*oauth2.Token, error) {
	// An alias can be reassigned. Only the current signed application ID selects its cache.
	key := s.organization + "/" + grant.ID + "/" + s.alias
	scopes := grant.Scopes
	child, err := s.cached(ctx, root.Applications[key], scopes)
	if err != nil {
		return nil, err
	}
	if child == nil {
		var err error
		child, err = s.authorize(ctx, root, scopes)
		if err != nil {
			return nil, err
		}
		if err := s.save(root, child, key, save); err != nil {
			return nil, err
		}
	}
	if !usable(child.MembershipToken) {
		child, err = s.refreshAndSave(ctx, root, child, scopes, key, save)
		if err != nil {
			return nil, err
		}
	}
	if s.endpoint != "" && child.StackURL != s.endpoint {
		return nil, errors.New("application endpoint changed; recreate the client")
	}
	if s.endpoint == "" {
		s.endpoint = child.StackURL
	}
	return cleanToken(child.MembershipToken), nil
}

func (s *applicationTransport) refreshAndSave(ctx context.Context, root, child *Session, scopes []string, key string, save func(*Session) error) (*Session, error) {
	renewed, refreshErr := s.refresh(ctx, child, scopes)
	if renewed != nil {
		if err := s.save(root, renewed, key, save); err != nil {
			return nil, err
		}
	}
	return renewed, refreshErr
}

func (s *applicationTransport) cached(ctx context.Context, cached *Session, scopes []string) (*Session, error) {
	child := copyRoot(cached)
	if child == nil {
		return nil, nil
	}
	matches, err := s.verify(ctx, child, scopes, true)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, nil
	}
	return child, nil
}

func (s *applicationTransport) save(root, child *Session, key string, save func(*Session) error) error {
	if root.Applications == nil {
		root.Applications = make(map[string]*Session)
	}
	root.Applications[key] = copyRoot(child)
	return s.manager.saveRoot(root, save)
}

func (s *applicationTransport) authorize(ctx context.Context, root *Session, scopes []string) (*Session, error) {
	m := s.manager
	config := *m.config
	config.Scopes = []string{"openid", "offline_access"}
	resource := applicationResource(s.alias, scopes)
	token, raw, err := deviceLogin(ctx, m.base, m.options, &config, m.out, m.open,
		[]oauth2.AuthCodeOption{oauth2.SetAuthURLParam("organization_id", s.organization), oauth2.SetAuthURLParam("id_token_hint", root.IDToken), oauth2.SetAuthURLParam("resource", resource)},
		[]oauth2.AuthCodeOption{oauth2.SetAuthURLParam("resource", resource)})
	if err != nil {
		return nil, err
	}
	child := &Session{Options: m.options, Application: s.alias, IDToken: raw, MembershipToken: cleanToken(token)}
	child.Options.Organization = s.organization
	matches, err := s.verify(ctx, child, scopes, false)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errors.New("application grant permissions differ from root identity")
	}
	return child, nil
}

func (s *applicationTransport) verify(ctx context.Context, child *Session, scopes []string, stored bool) (bool, error) {
	return s.verifyWithProofExpiry(ctx, child, scopes, stored, stored)
}

func (s *applicationTransport) verifyWithProofExpiry(ctx context.Context, child *Session, scopes []string, stored, storedProof bool) (bool, error) {
	m := s.manager
	expected := m.options
	expected.Organization = s.organization
	if child.Options != expected || child.Application != s.alias || child.StackToken != nil || child.IDToken == "" || child.MembershipToken == nil {
		return false, errors.New("invalid application session; log in again")
	}
	subject, proofScopes, err := s.verifyProof(ctx, child.IDToken, storedProof)
	if err != nil {
		return false, err
	}
	// Resource access tokens are issued to the application backend, not fctl.
	// The separate ID proof still requires fctl; custom audience validation
	// below accepts only a unique secure backend after signature/issuer checks.
	access, err := m.provider.Verifier(&oidc.Config{SkipClientIDCheck: true, SkipExpiryCheck: stored}).Verify(ctx, child.MembershipToken.AccessToken)
	if err != nil {
		return false, safeError(ctx, "verify application access", err)
	}
	var grant struct {
		Organization string `json:"organization_id"`
		Scope        string `json:"scope"`
	}
	if err := access.Claims(&grant); err != nil {
		return false, errors.New("invalid application access claims")
	}
	if grant.Organization != s.organization || access.Subject == "" || access.Subject != subject {
		return false, errors.New("application grant identity differs from root identity")
	}
	endpoint, err := applicationEndpoint(access.Audience, m.options.ClientID)
	if err != nil {
		return false, err
	}
	if child.StackURL != "" && child.StackURL != endpoint {
		return false, errors.New("application endpoint differs from signed audience")
	}
	if s.endpoint != "" && s.endpoint != endpoint {
		return false, errors.New("application endpoint changed; recreate the client")
	}
	child.StackURL = endpoint
	child.MembershipToken.Expiry = minExpiry(child.MembershipToken.Expiry, access.Expiry)
	if err := boundSessionTokens(child); err != nil {
		return false, err
	}
	// A permission change is a cache miss only after the stored proof and access
	// agree with each other and all signature, subject and endpoint checks pass.
	if !equalApplicationScopes(strings.Fields(grant.Scope), proofScopes) {
		return false, errors.New("application access permissions differ from signed identity proof")
	}
	return slices.Equal(proofScopes, scopes), nil
}

func (s *applicationTransport) verifyProof(ctx context.Context, raw string, stored bool) (string, []string, error) {
	m := s.manager
	_, id, err := verifyIdentity(ctx, m.provider, m.options, raw, stored)
	if err != nil {
		return "", nil, err
	}
	var proof struct {
		Resources []string `json:"resources"`
	}
	if err := id.Claims(&proof); err != nil {
		return "", nil, errors.New("invalid application identity proof")
	}
	if len(proof.Resources) != 1 {
		return "", nil, errors.New("application identity proof requires one resource")
	}
	_, granted, _ := strings.Cut(proof.Resources[0], "|")
	scopes := filterApplicationScopes(strings.Fields(granted))
	if len(scopes) == 0 || !matchesApplicationResource(proof.Resources[0], s.alias, scopes) {
		return "", nil, errors.New("application identity proof differs from requested resource")
	}
	_, rootID, err := verifyIdentity(ctx, m.provider, m.options, m.root.IDToken, true)
	if err != nil {
		return "", nil, err
	}
	if id.Expiry.IsZero() || id.Subject == "" || id.Subject != rootID.Subject {
		return "", nil, errors.New("application identity proof differs from root subject")
	}
	return id.Subject, scopes, nil
}

func matchesApplicationResource(resource, alias string, scopes []string) bool {
	target, granted, ok := strings.Cut(resource, "|")
	return ok && target == "app://"+alias && equalApplicationScopes(strings.Fields(granted), scopes)
}

func equalApplicationScopes(granted, expected []string) bool {
	filtered := filterApplicationScopes(granted)
	return len(filtered) == len(granted) && slices.Equal(filtered, expected)
}

func applicationEndpoint(audiences []string, clientID string) (string, error) {
	var endpoint string
	seenClient := false
	for _, audience := range audiences {
		if audience == clientID {
			if seenClient {
				return "", errors.New("duplicate application client audience")
			}
			seenClient = true
			continue
		}
		uri, err := url.Parse(audience)
		if err != nil || uri.Scheme != "https" || validateEndpoint(audience) != nil || uri.RawPath != "" || uri.Opaque != "" || uri.ForceQuery || strings.ContainsAny(uri.Path, "\\%\x00") || path.Clean(uri.Path) != strings.TrimRight(uri.Path, "/") && strings.TrimRight(uri.Path, "/") != "" || endpoint != "" {
			return "", errors.New("application access requires one secure backend audience")
		}
		endpoint = strings.TrimRight(audience, "/")
	}
	if endpoint == "" {
		return "", errors.New("application access is missing a secure backend audience")
	}
	return endpoint, nil
}

func (s *applicationTransport) refresh(ctx context.Context, child *Session, scopes []string) (*Session, error) {
	if child.MembershipToken.RefreshToken == "" {
		return nil, errors.New("application session expired; log in again")
	}
	m := s.manager
	config := *m.config
	config.Scopes = nil
	old := cleanToken(child.MembershipToken)
	old.Expiry = time.Now().Add(-time.Hour)
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, authClient(ctx, m.base, m.options.Issuer))
	fresh, err := config.TokenSource(authCtx, old).Token()
	if err != nil {
		return nil, safeError(ctx, "application refresh", err)
	}
	child.MembershipToken = cleanToken(fresh)
	// A refresh may omit id_token. The previously verified ID token remains
	// authorization evidence after expiry; fresh access still requires expiry.
	storedProof := true
	if raw, ok := fresh.Extra("id_token").(string); ok && raw != "" {
		child.IDToken = raw
		storedProof = false
	}
	if err := s.validateRefresh(ctx, child, scopes, storedProof); err != nil {
		child.IDToken = ""
		return child, err
	}
	return child, nil
}

func (s *applicationTransport) validateRefresh(ctx context.Context, child *Session, scopes []string, storedProof bool) error {
	if err := boundToken(child.MembershipToken, time.Now()); err != nil {
		return err
	}
	matches, err := s.verifyWithProofExpiry(ctx, child, scopes, false, storedProof)
	if err != nil {
		return err
	}
	if !matches {
		return errors.New("refreshed application permissions differ from root identity; log in again")
	}
	return nil
}

func (s *applicationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := s.route(req.URL); err != nil {
		return nil, err
	}
	if req.Host != "" && !strings.EqualFold(req.Host, req.URL.Host) {
		return nil, errors.New("application client refused a different request Host")
	}
	ctx, cancel := context.WithCancel(req.Context())
	//nolint:contextcheck // The client lifetime also cancels independent request contexts.
	stop := context.AfterFunc(s.ctx, cancel)
	cleanup := func() { stop(); cancel() }
	if err := s.ctx.Err(); err != nil {
		cleanup()
		return nil, err
	}
	token, err := s.token(ctx, req.Method)
	if err != nil {
		cleanup()
		return nil, err
	}
	transport := &oauth2.Transport{Source: oauth2.StaticTokenSource(token), Base: baseTransport(s.manager.base)}
	resp, err := transport.RoundTrip(req.Clone(ctx))
	if err != nil {
		cleanup()
		return nil, err
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cleanup: cleanup}
	return resp, nil
}

func (s *applicationTransport) route(uri *url.URL) error {
	if uri == nil || uri.User != nil || uri.RawPath != "" || uri.Opaque != "" || !sameOrigin(s.endpoint, uri.String()) || strings.ContainsAny(uri.Path, "\\%\x00") || path.Clean(uri.Path) != strings.TrimSuffix(uri.Path, "/") && uri.Path != "/" {
		return errors.New("application client refused an unsafe request URL")
	}
	endpoint, err := url.Parse(s.endpoint)
	if err != nil {
		return errors.New("invalid application endpoint")
	}
	prefix := strings.TrimRight(endpoint.Path, "/")
	if uri.Path != prefix && !strings.HasPrefix(uri.Path, prefix+"/") {
		return errors.New("application client refused a request outside its backend path")
	}
	return nil
}
