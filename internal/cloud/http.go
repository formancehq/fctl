package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/formancehq/fctl/v4/internal/api"
)

const requestTimeout = 30 * time.Second
const maxResponse = 1 << 20

type discovery struct {
	Issuer         string   `json:"issuer"`
	TokenEndpoint  string   `json:"token_endpoint"`
	DeviceEndpoint string   `json:"device_authorization_endpoint"`
	JWKS           string   `json:"jwks_uri"`
	UserInfo       string   `json:"userinfo_endpoint"`
	Algorithms     []string `json:"id_token_signing_alg_values_supported"`
}

func discover(ctx context.Context, base *http.Client, issuer string) (*discovery, error) {
	client := authClient(ctx, base, issuer)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, errors.New("invalid discovery URL")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, safeError(ctx, "OIDC discovery", err)
	}
	defer closeResponse(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery failed (HTTP %d)", resp.StatusCode)
	}
	var metadata discovery
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, errors.New("invalid OIDC discovery response")
	}
	if metadata.Issuer != issuer {
		return nil, errors.New("OIDC discovery issuer mismatch")
	}
	if err := validateEndpoint(metadata.TokenEndpoint); err != nil || !sameOrigin(issuer, metadata.TokenEndpoint) {
		return nil, errors.New("invalid or cross-origin OIDC token endpoint")
	}
	return &metadata, nil
}

func membership(ctx context.Context, base *http.Client, options Options) (*oidc.Provider, *oauth2.Config, error) {
	metadata, err := discover(ctx, base, options.Issuer)
	if err != nil {
		return nil, nil, err
	}
	for _, endpoint := range []string{metadata.DeviceEndpoint, metadata.JWKS} {
		if err := validateEndpoint(endpoint); err != nil || !sameOrigin(options.Issuer, endpoint) {
			return nil, nil, errors.New("invalid or cross-origin Membership endpoint")
		}
	}
	if metadata.UserInfo != "" {
		if err := validateEndpoint(metadata.UserInfo); err != nil || !sameOrigin(options.Issuer, metadata.UserInfo) {
			return nil, nil, errors.New("invalid or cross-origin Membership userinfo endpoint")
		}
	}
	client := authClient(ctx, base, options.Issuer)
	// Membership's historical device endpoint authenticates the public client
	// using Basic auth. DeviceAuth otherwise sends client_id in the form.
	client.Transport = &deviceTransport{base: client.Transport, endpoint: metadata.DeviceEndpoint, clientID: options.ClientID}
	provider := (&oidc.ProviderConfig{IssuerURL: metadata.Issuer, TokenURL: metadata.TokenEndpoint, DeviceAuthURL: metadata.DeviceEndpoint, JWKSURL: metadata.JWKS, UserInfoURL: metadata.UserInfo, Algorithms: metadata.Algorithms}).NewProvider(oidc.ClientContext(ctx, client))
	config := &oauth2.Config{ClientID: options.ClientID, Scopes: []string{oidc.ScopeOpenID, "offline_access"}, Endpoint: oauth2.Endpoint{TokenURL: metadata.TokenEndpoint, DeviceAuthURL: metadata.DeviceEndpoint, AuthStyle: oauth2.AuthStyleInHeader}}
	return provider, config, nil
}

type deviceTransport struct {
	base               http.RoundTripper
	endpoint, clientID string
}

func (t *deviceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() == t.endpoint && req.Method == http.MethodPost {
		req = req.Clone(req.Context())
		req.SetBasicAuth(t.clientID, "")
	}
	return t.base.RoundTrip(req)
}

func exchange(ctx context.Context, base *http.Client, stackURL, assertion string) (*oauth2.Token, error) {
	metadata, err := discover(ctx, base, stackURL+"/api/auth")
	if err != nil {
		return nil, err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assertion}, "scope": {"openid email"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.New("invalid stack token endpoint")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	now := time.Now()
	resp, err := authClient(ctx, base, stackURL).Do(req)
	if err != nil {
		return nil, safeError(ctx, "stack token exchange", err)
	}
	defer closeResponse(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, authenticationFailure(fmt.Sprintf("stack token exchange failed (HTTP %d)", resp.StatusCode))
	}
	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.New("invalid stack token response")
	}
	if result.ExpiresIn <= 0 {
		return nil, errors.New("stack token response is missing a positive expires_in")
	}
	seconds := min(result.ExpiresIn, int64(maxTokenLifetime/time.Second))
	token := &oauth2.Token{AccessToken: result.AccessToken, TokenType: result.TokenType, Expiry: now.Add(time.Duration(seconds) * time.Second)}
	if err := boundToken(token, time.Now()); err != nil {
		return nil, err
	}
	return token, nil
}

func validateEndpoint(endpoint string) error {
	if err := api.ValidateURL(endpoint); err != nil {
		return err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("invalid endpoint")
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return errors.New("cloud endpoints require HTTPS (except loopback)")
		}
	}
	if strings.ContainsAny(endpoint, "\r\n\x1b") {
		return errors.New("invalid endpoint")
	}
	return nil
}
func sameOrigin(a, b string) bool {
	ua, ea := url.Parse(a)
	ub, eb := url.Parse(b)
	return ea == nil && eb == nil && ua.Host != "" && strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}
func cloneClient(base *http.Client) *http.Client {
	if base == nil {
		base = http.DefaultClient
	}
	clone := *base
	if clone.Timeout == 0 {
		clone.Timeout = requestTimeout
	}
	return &clone
}
func baseTransport(base *http.Client) http.RoundTripper {
	if base != nil && base.Transport != nil {
		return base.Transport
	}
	return http.DefaultTransport
}
func noRedirect(_ *http.Request, _ []*http.Request) error {
	return errors.New("cloud HTTP redirects are disabled")
}
func authClient(ctx context.Context, base *http.Client, origin string) *http.Client {
	client := cloneClient(base)
	timeout := min(client.Timeout, requestTimeout)
	// Enforce this inside RoundTrip: DeviceAuth does not close its response,
	// which would leave http.Client's cancellation timer alive until expiry.
	client.Timeout = 0
	client.CheckRedirect = noRedirect
	client.Transport = &authTransport{ctx: ctx, base: baseTransport(base), origin: origin, timeout: timeout}
	return client
}

type authTransport struct {
	ctx     context.Context
	base    http.RoundTripper
	origin  string
	timeout time.Duration
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if validateEndpoint(req.URL.String()) != nil || !sameOrigin(t.origin, req.URL.String()) {
		return nil, errors.New("authentication request refused an invalid or cross-origin endpoint")
	}
	// oauth2.DeviceAuth does not attach its supplied context to the request.
	// Bound requests here also constrain background JWKS fetches by coreos.
	ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
	//nolint:contextcheck // Bind library background/device requests to the caller lifetime as well as the request context.
	stop := context.AfterFunc(t.ctx, cancel)
	defer cancel()
	defer stop()
	if err := t.ctx.Err(); err != nil {
		return nil, err
	}
	resp, err := t.base.RoundTrip(req.Clone(ctx))
	if err != nil {
		return nil, safeError(ctx, "authentication HTTP request", err)
	}
	originalBody := resp.Body
	defer closeResponse(originalBody)
	body, err := io.ReadAll(io.LimitReader(originalBody, maxResponse+1))
	if err != nil {
		return nil, safeError(ctx, "authentication response", err)
	}
	if len(body) > maxResponse {
		return nil, errors.New("authentication response exceeds size limit")
	}
	// Buffer and close all auth responses, including DeviceAuth responses (the
	// upstream library does not close these). Never expose their bodies in errors.
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func safeError(ctx context.Context, operation string, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w", operation, ctx.Err())
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", operation, context.DeadlineExceeded)
	}
	if trusted, ok := errors.AsType[*authenticationError](err); ok {
		return authenticationFailure(operation + ": " + trusted.Error())
	}
	if retrieve, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		suffix := safeOAuthCode(retrieve.ErrorCode)
		if retrieve.Response != nil {
			return authenticationFailure(fmt.Sprintf("%s failed (HTTP %d)%s", operation, retrieve.Response.StatusCode, suffix))
		}
		return authenticationFailure(fmt.Sprintf("%s failed%s", operation, suffix))
	}
	return authenticationFailure(operation + " failed")
}

// Only package-generated labels/status/code enter this type. Never retain the
// original provider error, whose body or description can contain credentials.
type authenticationError struct{ message string }

func (e *authenticationError) Error() string { return e.message }

func authenticationFailure(message string) error { return &authenticationError{message: message} }

func safeOAuthCode(code string) string {
	switch code {
	case "invalid_grant", "invalid_client", "invalid_scope", "invalid_request",
		"access_denied", "expired_token", "unauthorized_client", "unsupported_grant_type",
		"authorization_pending", "slow_down", "server_error", "temporarily_unavailable":
		return ": " + code
	default:
		return ""
	}
}

// cancelBody keeps the client lifetime and request cancellation active through
// streaming reads, then releases the context hook when the caller closes it.
type cancelBody struct {
	io.ReadCloser
	cleanup func()
}

func (b *cancelBody) Close() error {
	defer b.cleanup()
	return b.ReadCloser.Close()
}

// The response is already consumed or rejected; a close error cannot change the
// authentication outcome. Discard it without logging response or credentials.
func closeResponse(body io.Closer) {
	if err := body.Close(); err != nil {
		return
	}
}
