package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

func TestNoneAuthDoesNotSetAuthorization(t *testing.T) {
	server := authHeaderServer(t, "")
	defer server.Close()

	client, err := NewHTTPClient(context.Background(), v4config.Auth{Method: v4config.AuthMethodNone}, nil, Options{})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	if _, err := client.Get(server.URL); err != nil {
		t.Fatalf("request: %v", err)
	}
}

func TestTokenAuthFromCredentialRef(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	if err := store.Set(ctx, "token-ref", "abc123"); err != nil {
		t.Fatalf("set token: %v", err)
	}
	server := authHeaderServer(t, "Bearer abc123")
	defer server.Close()

	client, err := NewHTTPClient(ctx, v4config.Auth{
		Method:   v4config.AuthMethodToken,
		TokenRef: "token-ref",
	}, store, Options{})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	if _, err := client.Get(server.URL); err != nil {
		t.Fatalf("request: %v", err)
	}
}

func TestTokenAuthFromEnvRef(t *testing.T) {
	source, err := NewTokenSource(v4config.Auth{
		Method:   v4config.AuthMethodToken,
		TokenRef: "env://FCTL_TOKEN",
	}, nil, Options{Env: func(key string) string {
		if key != "FCTL_TOKEN" {
			t.Fatalf("unexpected env key %q", key)
		}
		return "from-env"
	}})
	if err != nil {
		t.Fatalf("new token source: %v", err)
	}
	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if token.AccessToken != "from-env" || token.TokenType != "Bearer" {
		t.Fatalf("unexpected token: %#v", token)
	}
}

func TestCloudDeviceAuthFromCredentialRef(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	encoded, err := MarshalDeviceTokens(DeviceTokens{
		AccessToken: StoredAccessToken{
			Token:     "device-token",
			TokenType: "Bearer",
		},
	})
	if err != nil {
		t.Fatalf("marshal device tokens: %v", err)
	}
	if err := store.Set(ctx, "root-token-ref", encoded); err != nil {
		t.Fatalf("set device tokens: %v", err)
	}
	server := authHeaderServer(t, "Bearer device-token")
	defer server.Close()

	client, err := NewHTTPClient(ctx, v4config.Auth{
		Method:   v4config.AuthMethodCloudDevice,
		TokenRef: "root-token-ref",
	}, store, Options{})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	if _, err := client.Get(server.URL); err != nil {
		t.Fatalf("request: %v", err)
	}
}

func TestCloudDeviceAuthReadsMigratedV3RootTokens(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	if err := store.Set(ctx, "root-token-ref", `{"access":{"token":"migrated-token"},"id":{"token":"id-token"}}`); err != nil {
		t.Fatalf("set migrated tokens: %v", err)
	}

	source, err := NewTokenSource(v4config.Auth{
		Method:   v4config.AuthMethodCloudDevice,
		TokenRef: "root-token-ref",
	}, store, Options{})
	if err != nil {
		t.Fatalf("new token source: %v", err)
	}
	token, err := source.Token(ctx)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if token.AccessToken != "migrated-token" || token.TokenType != "Bearer" {
		t.Fatalf("unexpected token: %#v", token)
	}
}

func TestCloudDeviceAuthRefreshesWhenScopesAreMissing(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	encoded, err := MarshalDeviceTokens(DeviceTokens{
		AccessToken: StoredAccessToken{
			Token:        authTestJWT(t, map[string]any{"scope": "openid", "exp": time.Now().Add(time.Hour).Unix()}),
			TokenType:    "Bearer",
			RefreshToken: "refresh-token",
		},
		RefreshToken: "refresh-token",
	})
	if err != nil {
		t.Fatalf("marshal device tokens: %v", err)
	}
	if err := store.Set(ctx, "root-token-ref", encoded); err != nil {
		t.Fatalf("set device tokens: %v", err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, server.URL+"/token")
		case "/token":
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != DeviceClientID || clientSecret != "" {
				t.Fatalf("unexpected basic auth: %q %q %v", clientID, clientSecret, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "refresh_token" {
				t.Fatalf("unexpected grant_type %q", r.Form.Get("grant_type"))
			}
			if r.Form.Get("refresh_token") != "refresh-token" {
				t.Fatalf("unexpected refresh_token %q", r.Form.Get("refresh_token"))
			}
			if r.Form.Get("scope") != "organization:ListStacks" {
				t.Fatalf("unexpected scope %q", r.Form.Get("scope"))
			}
			fmt.Fprint(w, `{"access_token":"scoped-token","token_type":"Bearer","refresh_token":"next-refresh-token","expires_in":3600}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	source, err := NewTokenSource(v4config.Auth{
		Method:    v4config.AuthMethodCloudDevice,
		IssuerURL: server.URL,
		TokenRef:  "root-token-ref",
		Scopes:    []string{"organization:ListStacks"},
	}, store, Options{})
	if err != nil {
		t.Fatalf("new token source: %v", err)
	}
	token, err := source.Token(ctx)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if token.AccessToken != "scoped-token" {
		t.Fatalf("unexpected token: %#v", token)
	}
	stored, err := store.Get(ctx, "root-token-ref")
	if err != nil {
		t.Fatalf("get stored refreshed token: %v", err)
	}
	if !strings.Contains(stored, "next-refresh-token") {
		t.Fatalf("expected refreshed token to be stored, got %s", stored)
	}
}

func TestClientCredentialsAuth(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	if err := store.Set(ctx, "secret-ref", "secret"); err != nil {
		t.Fatalf("set secret: %v", err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, server.URL+"/token")
		case "/token":
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != "client" || clientSecret != "secret" {
				t.Fatalf("unexpected basic auth: %q %q %v", clientID, clientSecret, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "client_credentials" {
				t.Fatalf("unexpected grant_type %q", r.Form.Get("grant_type"))
			}
			if r.Form.Get("scope") != "organization:Read organization:ListStacks" {
				t.Fatalf("unexpected scope %q", r.Form.Get("scope"))
			}
			fmt.Fprint(w, `{"access_token":"cc-token","token_type":"Bearer"}`)
		case "/resource":
			if got := r.Header.Get("Authorization"); got != "Bearer cc-token" {
				t.Fatalf("unexpected authorization header %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(ctx, v4config.Auth{
		Method:    v4config.AuthMethodClientCredentials,
		IssuerURL: server.URL,
		ClientID:  "client",
		SecretRef: "secret-ref",
		Scopes:    []string{"organization:Read", "organization:ListStacks"},
	}, store, Options{})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	rsp, err := client.Get(server.URL + "/resource")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rsp.StatusCode)
	}
}

func authHeaderServer(t *testing.T, expected string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != expected {
			t.Fatalf("expected authorization header %q, got %q", expected, got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestTokenAuthFromStdinRef(t *testing.T) {
	source, err := NewTokenSource(v4config.Auth{
		Method:   v4config.AuthMethodToken,
		TokenRef: "stdin://",
	}, nil, Options{Stdin: strings.NewReader("from-stdin\n")})
	if err != nil {
		t.Fatalf("new token source: %v", err)
	}
	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if token.AccessToken != "from-stdin" {
		t.Fatalf("unexpected token: %#v", token)
	}
}

func authTestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "none"})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal jwt payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}
