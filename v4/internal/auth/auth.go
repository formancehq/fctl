// Package auth resolves v4 target-local authentication into HTTP clients.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

type Options struct {
	HTTPClient *http.Client
	Env        func(string) string
	Stdin      io.Reader
}

type Token struct {
	AccessToken string
	TokenType   string
}

type TokenSource interface {
	Token(ctx context.Context) (Token, error)
}

func NewHTTPClient(ctx context.Context, authConfig v4config.Auth, store credentials.Store, options Options) (*http.Client, error) {
	base := options.HTTPClient
	if base == nil {
		base = http.DefaultClient
	}

	source, err := NewTokenSource(authConfig, store, options)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return base, nil
	}

	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport:     &bearerTransport{source: source, base: transport},
		CheckRedirect: base.CheckRedirect,
		Jar:           base.Jar,
		Timeout:       base.Timeout,
	}, nil
}

func NewTokenSource(authConfig v4config.Auth, store credentials.Store, options Options) (TokenSource, error) {
	switch authConfig.Method {
	case v4config.AuthMethodNone:
		return nil, nil
	case v4config.AuthMethodToken:
		return tokenSourceForRef(authConfig.TokenRef, store, options)
	case v4config.AuthMethodClientCredentials:
		if store == nil {
			return nil, errors.New("credential store is required for client_credentials auth")
		}
		httpClient := options.HTTPClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		return &ClientCredentialsSource{
			IssuerURL:  authConfig.IssuerURL,
			ClientID:   authConfig.ClientID,
			SecretRef:  authConfig.SecretRef,
			Store:      store,
			HTTPClient: httpClient,
		}, nil
	case v4config.AuthMethodCloudDevice, v4config.AuthMethodOIDCDevice:
		return nil, fmt.Errorf("auth method %q is not implemented yet", authConfig.Method)
	default:
		return nil, fmt.Errorf("unsupported auth method %q", authConfig.Method)
	}
}

type StaticTokenSource struct {
	TokenValue Token
}

func (s StaticTokenSource) Token(context.Context) (Token, error) {
	return normalizeToken(s.TokenValue), nil
}

type CredentialTokenSource struct {
	Ref   string
	Store credentials.Store
}

func (s CredentialTokenSource) Token(ctx context.Context) (Token, error) {
	if s.Store == nil {
		return Token{}, errors.New("credential store is required")
	}
	value, err := s.Store.Get(ctx, s.Ref)
	if err != nil {
		return Token{}, err
	}
	return normalizeToken(Token{AccessToken: value, TokenType: "Bearer"}), nil
}

type ClientCredentialsSource struct {
	IssuerURL  string
	ClientID   string
	SecretRef  string
	Store      credentials.Store
	HTTPClient *http.Client

	mu          sync.Mutex
	cachedToken *Token
	tokenURL    string
}

func (s *ClientCredentialsSource) Token(ctx context.Context) (Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedToken != nil {
		return *s.cachedToken, nil
	}
	if s.HTTPClient == nil {
		s.HTTPClient = http.DefaultClient
	}
	if s.Store == nil {
		return Token{}, errors.New("credential store is required")
	}
	if s.IssuerURL == "" {
		return Token{}, errors.New("issuer URL is required")
	}
	if s.ClientID == "" {
		return Token{}, errors.New("client ID is required")
	}

	secret, err := s.Store.Get(ctx, s.SecretRef)
	if err != nil {
		return Token{}, err
	}
	tokenURL, err := s.resolveTokenURL(ctx)
	if err != nil {
		return Token{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.SetBasicAuth(s.ClientID, secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rsp, err := s.HTTPClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))
		return Token{}, fmt.Errorf("client credentials token request failed: status %d: %s", rsp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&tokenResponse); err != nil {
		return Token{}, fmt.Errorf("decode token response: %w", err)
	}
	token := normalizeToken(Token{
		AccessToken: tokenResponse.AccessToken,
		TokenType:   tokenResponse.TokenType,
	})
	if token.AccessToken == "" {
		return Token{}, errors.New("token response did not include access_token")
	}
	s.cachedToken = &token
	return token, nil
}

func (s *ClientCredentialsSource) resolveTokenURL(ctx context.Context) (string, error) {
	if s.tokenURL != "" {
		return s.tokenURL, nil
	}

	discoveryURL := strings.TrimRight(s.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", err
	}
	rsp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))
		return "", fmt.Errorf("oidc discovery failed: status %d: %s", rsp.StatusCode, string(body))
	}

	var discovery struct {
		TokenEndpoint string `json:"token_endpoint"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&discovery); err != nil {
		return "", fmt.Errorf("decode oidc discovery: %w", err)
	}
	if discovery.TokenEndpoint == "" {
		return "", errors.New("oidc discovery did not include token_endpoint")
	}
	s.tokenURL = discovery.TokenEndpoint
	return s.tokenURL, nil
}

type bearerTransport struct {
	source TokenSource
	base   http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.Token(req.Context())
	if err != nil {
		return nil, err
	}

	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	cloned.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	return t.base.RoundTrip(cloned)
}

func tokenSourceForRef(ref string, store credentials.Store, options Options) (TokenSource, error) {
	switch {
	case ref == "stdin://":
		if options.Stdin == nil {
			return nil, errors.New("stdin token source requires stdin")
		}
		data, err := io.ReadAll(options.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read token from stdin: %w", err)
		}
		return StaticTokenSource{TokenValue: Token{AccessToken: strings.TrimSpace(string(data)), TokenType: "Bearer"}}, nil
	case strings.HasPrefix(ref, "env://"):
		getenv := options.Env
		if getenv == nil {
			getenv = os.Getenv
		}
		name := strings.TrimPrefix(ref, "env://")
		value := strings.TrimSpace(getenv(name))
		if value == "" {
			return nil, fmt.Errorf("environment variable %s is empty", name)
		}
		return StaticTokenSource{TokenValue: Token{AccessToken: value, TokenType: "Bearer"}}, nil
	default:
		return CredentialTokenSource{Ref: ref, Store: store}, nil
	}
}

func normalizeToken(token Token) Token {
	token.AccessToken = strings.TrimSpace(token.AccessToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	return token
}
