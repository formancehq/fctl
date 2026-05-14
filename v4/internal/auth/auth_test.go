package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
