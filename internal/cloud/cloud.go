// Package cloud authenticates a user through Membership and exchanges the
// stack-scoped Membership token for a data-plane token. Sessions contain secrets
// and must be persisted by the caller with restrictive permissions.
package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const refreshMargin = 30 * time.Second
const maxTokenLifetime = 24 * time.Hour

// Options identifies the Membership issuer and the one stack to authorize.
type Options struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	Organization string `json:"organization"`
	Stack        string `json:"stack"`
}

// Session is serializable authentication state. Do not log it. StackURL is
// checked against signed Membership claims when constructing a client.
type Session struct {
	Options         Options       `json:"options"`
	IDToken         string        `json:"id_token"`
	MembershipToken *oauth2.Token `json:"membership_token"`
	StackURL        string        `json:"stack_url"`
	StackToken      *oauth2.Token `json:"stack_token,omitzero"`
}

// Coordinator serializes authentication for one saved connection. While holding
// its authentication lock, it reloads the current session and invokes work with
// that session and a revision-checked save function. The lock must cover work,
// including all refresh, exchange and save operations. Logout and replacement
// can independently invalidate that save through the caller's revision check.
// The coordinator must invoke work exactly once and return its result.
type Coordinator func(context.Context, func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error)

func (o Options) normalized() (Options, error) {
	o.Issuer = strings.TrimRight(o.Issuer, "/")
	if o.ClientID == "" {
		o.ClientID = "fctl"
	}
	if err := validateEndpoint(o.Issuer); err != nil {
		return o, err
	}
	for _, id := range []string{o.ClientID, o.Organization, o.Stack} {
		if id == "" || strings.ContainsAny(id, " /|\t\r\n?#\\") {
			return o, errors.New("cloud requires a client ID, organization and stack with valid identifiers")
		}
	}
	return o, nil
}

func (o Options) resource() string {
	return "stack://" + o.Organization + "/" + o.Stack + "|stack:Read stack:Write"
}

// Login displays the device verification URI and user code on out (normally
// stderr). It never opens a browser or requests organization-wide permissions.
func Login(ctx context.Context, httpClient *http.Client, options Options, out io.Writer) (*Session, error) {
	options, err := options.normalized()
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("device authorization requires an output writer")
	}
	provider, config, err := membership(ctx, httpClient, options)
	if err != nil {
		return nil, err
	}
	deviceClient := authClient(ctx, httpClient, options.Issuer)
	deviceClient.Transport = &deviceTransport{base: deviceClient.Transport, endpoint: config.Endpoint.DeviceAuthURL, clientID: options.ClientID}
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, deviceClient)
	device, err := config.DeviceAuth(authCtx, oauth2.SetAuthURLParam("organization_id", options.Organization), oauth2.SetAuthURLParam("resource", options.resource()))
	if err != nil {
		return nil, safeError(ctx, "device authorization", err)
	}
	if device.DeviceCode == "" || device.UserCode == "" || validateEndpoint(device.VerificationURI) != nil || device.Interval < 0 || device.Interval > 3600 || !device.Expiry.After(time.Now()) || device.Expiry.After(time.Now().Add(time.Hour)) {
		return nil, errors.New("invalid device authorization response")
	}
	if strings.ContainsAny(device.UserCode, "\r\n\x1b") {
		return nil, errors.New("invalid device user code")
	}
	if _, err := fmt.Fprintf(out, "Open %s and enter code %s\n", device.VerificationURI, device.UserCode); err != nil {
		return nil, errors.New("cannot display device authorization")
	}
	token, err := config.DeviceAccessToken(authCtx, device, oauth2.SetAuthURLParam("resource", options.resource()))
	if err != nil {
		return nil, safeError(ctx, "device token", err)
	}
	if err := boundToken(token, time.Now()); err != nil {
		return nil, err
	}
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, errors.New("membership token response is missing an ID token")
	}
	stackURL, err := verifiedStack(ctx, provider, options, idToken, false)
	if err != nil {
		return nil, err
	}
	return &Session{Options: options, IDToken: idToken, MembershipToken: cleanToken(token), StackURL: stackURL}, nil
}

// Client validates the session, refreshes Membership when needed and exchanges
// its assertion for a stack token. The transport repeats this only on expiry;
// it never retries service operations after authentication errors.
// An optional coordinator surrounds every token acquisition, including the
// eager acquisition here and cached-token reads by subsequent requests. Its
// save function replaces save. After a coordinated save failure this client
// stops; recreating it requires the parent to resolve the saved session state.
// The caller must not mutate session concurrently with the returned client.
func Client(ctx context.Context, httpClient *http.Client, session *Session, save func(*Session) error, coordinators ...Coordinator) (*http.Client, string, error) {
	if len(coordinators) > 1 || (len(coordinators) == 1 && coordinators[0] == nil) {
		return nil, "", errors.New("cloud client requires at most one non-nil coordinator")
	}
	if session == nil || session.MembershipToken == nil {
		return nil, "", errors.New("missing cloud session; log in again")
	}
	options, err := session.Options.normalized()
	if err != nil {
		return nil, "", err
	}
	provider, config, err := membership(ctx, httpClient, options)
	if err != nil {
		return nil, "", err
	}
	state, err := validatedSession(ctx, provider, options, session)
	if err != nil {
		return nil, "", err
	}
	stackURL := state.StackURL
	source := &sessionTransport{ctx: ctx, base: httpClient, state: state, session: session, save: save, provider: provider, config: config, origin: stackURL}
	if len(coordinators) == 1 {
		source.coordinator = coordinators[0]
	}
	if _, err := source.token(ctx); err != nil {
		return nil, "", err
	}
	client := cloneClient(httpClient)
	client.Transport = source
	client.CheckRedirect = noRedirect
	return client, stackURL, nil
}

type sessionTransport struct {
	mu          sync.Mutex
	ctx         context.Context
	base        *http.Client
	state       Session
	session     *Session
	save        func(*Session) error
	provider    *oidc.Provider
	config      *oauth2.Config
	origin      string
	pendingSave bool
	coordinator Coordinator
}

func (s *sessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !sameOrigin(s.origin, req.URL.String()) || req.URL.User != nil {
		return nil, errors.New("cloud client refused a request outside its stack origin")
	}
	ctx, cancel := context.WithCancel(req.Context())
	//nolint:contextcheck // The client lifetime must cancel operations with independent request contexts.
	stop := context.AfterFunc(s.ctx, cancel)
	if err := s.ctx.Err(); err != nil {
		stop()
		cancel()
		return nil, err
	}
	token, err := s.token(ctx)
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	transport := &oauth2.Transport{Source: oauth2.StaticTokenSource(token), Base: baseTransport(s.base)}
	resp, err := transport.RoundTrip(req.Clone(ctx))
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cleanup: func() { stop(); cancel() }}
	return resp, nil
}

func (s *sessionTransport) token(ctx context.Context) (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		token, err := s.coordinator(ctx, func(current *Session, save func(*Session) error) (*oauth2.Token, error) {
			return s.coordinatedToken(ctx, current, save)
		})
		if err != nil {
			return nil, safeError(ctx, "coordinate cloud authentication", err)
		}
		return token, nil
	}
	return s.tokenLocked(ctx)
}

// token already holds mu; the coordinator holds the connection authentication
// lock before entering here. Never reacquire mu from its callback.
func (s *sessionTransport) coordinatedToken(ctx context.Context, current *Session, save func(*Session) error) (*oauth2.Token, error) {
	if s.pendingSave {
		// The in-memory credentials may contain a consumed/rotated refresh token.
		// A failed CAS can mean logout or replacement. Never reload an older token
		// or retry a save against a new revision; this client must stop.
		return nil, errors.New("cloud session save failed; recreate the client or log in again")
	}
	if save == nil {
		return nil, errors.New("cloud coordinator requires a session save function")
	}
	state, err := validatedSession(ctx, s.provider, s.state.Options, current)
	if err != nil {
		return nil, err
	}
	if state.StackURL != s.origin {
		return nil, errors.New("coordinated cloud session targets a different stack URL")
	}
	s.state = state
	*s.session = state
	s.save = save
	return s.tokenLocked(ctx)
}

func (s *sessionTransport) tokenLocked(ctx context.Context) (*oauth2.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.pendingSave {
		if err := s.persist(s.state); err != nil {
			return nil, err
		}
	}
	next := s.state
	changed := false
	if !usable(next.MembershipToken) {
		if next.MembershipToken.RefreshToken == "" {
			return nil, errors.New("cloud session expired; log in again")
		}
		refreshed, err := s.refresh(ctx, next)
		if err != nil {
			return nil, err
		}
		next = *refreshed
		changed = true
	}
	if !usable(next.StackToken) {
		exchanged, err := s.exchange(ctx, next, changed)
		if err != nil {
			return nil, err
		}
		next = *exchanged
		changed = true
	}
	if changed {
		if err := s.persist(next); err != nil {
			return nil, err
		}
	}
	return cleanToken(s.state.StackToken), nil
}

func (s *sessionTransport) refresh(ctx context.Context, next Session) (*Session, error) {
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, authClient(ctx, s.base, next.Options.Issuer))
	// Force refresh within our early-expiry margin. TokenSource preserves an
	// existing refresh token when the provider omits it from the response.
	old := cleanToken(next.MembershipToken)
	old.Expiry = time.Now().Add(-time.Hour)
	fresh, err := s.config.TokenSource(authCtx, old).Token()
	if err != nil {
		return nil, safeError(ctx, "Membership refresh", err)
	}
	if err := boundToken(fresh, time.Now()); err != nil {
		return nil, err
	}
	if raw, ok := fresh.Extra("id_token").(string); ok && raw != "" {
		uri, err := verifiedStack(ctx, s.provider, next.Options, raw, false)
		if err != nil {
			return nil, err
		}
		if uri != next.StackURL {
			return nil, errors.New("stack URL changed during refresh; log in again")
		}
		next.IDToken = raw
	}
	next.MembershipToken = cleanToken(fresh)
	next.StackToken = nil
	return &next, nil
}

func (s *sessionTransport) exchange(ctx context.Context, next Session, refreshed bool) (*Session, error) {
	token, err := exchange(ctx, s.base, next.StackURL, next.MembershipToken.AccessToken)
	if err != nil {
		// Preserve rotated refresh credentials even if the stack is unavailable.
		if refreshed {
			if saveErr := s.persist(next); saveErr != nil {
				return nil, saveErr
			}
		}
		return nil, err
	}
	token.Expiry = minExpiry(token.Expiry, next.MembershipToken.Expiry)
	next.StackToken = token
	return &next, nil
}

func (s *sessionTransport) persist(next Session) error {
	// Keep rotated credentials in memory even when persistence fails.
	s.state = next
	*s.session = next
	if s.save != nil {
		s.pendingSave = true
		snapshot := next
		snapshot.MembershipToken = cleanToken(next.MembershipToken)
		snapshot.StackToken = cleanToken(next.StackToken)
		if err := s.save(&snapshot); err != nil {
			return errors.New("cannot save cloud session")
		}
		s.pendingSave = false
	}
	return nil
}

func usable(t *oauth2.Token) bool {
	return t != nil && t.AccessToken != "" && !t.Expiry.IsZero() && t.Expiry.After(time.Now().Add(refreshMargin))
}
func cleanToken(t *oauth2.Token) *oauth2.Token {
	if t == nil {
		return nil
	}
	return &oauth2.Token{AccessToken: t.AccessToken, TokenType: t.TokenType, RefreshToken: t.RefreshToken, Expiry: t.Expiry}
}
func boundToken(t *oauth2.Token, now time.Time) error {
	if t == nil || t.AccessToken == "" || t.Expiry.IsZero() || !t.Expiry.After(now) || (t.TokenType != "" && !strings.EqualFold(t.TokenType, "Bearer")) || strings.ContainsAny(t.AccessToken, "\r\n") {
		return errors.New("invalid or expired OAuth token")
	}
	t.TokenType = "Bearer"
	t.Expiry = minExpiry(t.Expiry, now.Add(maxTokenLifetime))
	return nil
}
func minExpiry(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func verifiedStack(ctx context.Context, provider *oidc.Provider, options Options, raw string, stored bool) (string, error) {
	token, err := provider.Verifier(&oidc.Config{ClientID: options.ClientID, SkipExpiryCheck: stored}).Verify(ctx, raw)
	if err != nil {
		return "", safeError(ctx, "verify Membership identity", err)
	}
	var claims identityClaims
	if err := token.Claims(&claims); err != nil {
		return "", errors.New("invalid Membership organization claims")
	}
	return selectStack(options, claims)
}

type stackAccess struct {
	ID     string   `json:"id"`
	URI    string   `json:"uri"`
	Scopes []string `json:"scopes"`
}
type organizationAccess struct {
	ID     string        `json:"id"`
	Stacks []stackAccess `json:"stacks"`
}
type identityClaims struct {
	Organizations []organizationAccess `json:"org"`
}

func selectStack(options Options, claims identityClaims) (string, error) {
	for _, organization := range claims.Organizations {
		if organization.ID != options.Organization {
			continue
		}
		index := slices.IndexFunc(organization.Stacks, func(stack stackAccess) bool { return stack.ID == options.Stack })
		if index < 0 {
			break
		}
		stack := organization.Stacks[index]
		if !slices.Contains(stack.Scopes, "stack:Read") && !slices.Contains(stack.Scopes, "stack:Write") {
			return "", errors.New("identity grants no access to the requested stack")
		}
		uri := strings.TrimRight(stack.URI, "/")
		if err := validateEndpoint(uri); err != nil {
			return "", errors.New("identity contains an invalid stack URL")
		}
		return uri, nil
	}
	return "", errors.New("identity does not authorize the requested organization and stack")
}

// Stored ID tokens prove the original authorization after expiry. Fresh ID
// tokens received by Login or refresh still require a valid expiry.
func validatedSession(ctx context.Context, provider *oidc.Provider, expected Options, session *Session) (Session, error) {
	if session == nil || session.MembershipToken == nil {
		return Session{}, errors.New("missing cloud session; log in again")
	}
	options, err := session.Options.normalized()
	if err != nil {
		return Session{}, err
	}
	if options != expected {
		return Session{}, errors.New("cloud session options changed; recreate the client")
	}
	stackURL, err := verifiedStack(ctx, provider, options, session.IDToken, true)
	if err != nil {
		return Session{}, err
	}
	if session.StackURL != stackURL {
		return Session{}, errors.New("session stack URL differs from verified claims; log in again")
	}
	state := *session
	state.Options = options
	state.MembershipToken = cleanToken(session.MembershipToken)
	state.StackToken = cleanToken(session.StackToken)
	if err := boundSessionTokens(&state); err != nil {
		return Session{}, err
	}
	return state, nil
}

func boundSessionTokens(state *Session) error {
	if err := boundToken(state.MembershipToken, time.Now()); err != nil {
		if state.MembershipToken.RefreshToken == "" {
			return err
		}
		state.MembershipToken.Expiry = time.Time{}
	}
	if state.StackToken != nil {
		if err := boundToken(state.StackToken, time.Now()); err != nil {
			state.StackToken = nil
		}
	}
	return nil
}
