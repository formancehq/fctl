package runtime

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

func TestNewResolvesCurrentStackContext(t *testing.T) {
	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth:     config.Auth{Method: config.AuthMethodNone},
				API:      map[string]string{"ledger": string(config.APIPolicyPinned)},
			},
		},
	})

	rt, err := New(context.Background(), Options{
		ConfigPath:      configPath,
		Credentials:     credentials.NewMemoryStore(),
		Manifest:        capabilities.Manifest{SpecVersion: "test"},
		Compatibility:   capabilities.ComponentCompatibility{},
		ContextOverride: config.ContextOverride{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if rt.ContextName != "local" {
		t.Fatalf("expected local context, got %q", rt.ContextName)
	}
	if rt.Target.Kind != TargetKindStack || rt.Target.URL != "http://localhost/api" {
		t.Fatalf("unexpected target: %#v", rt.Target)
	}
	if policy := rt.APIPolicyFor("ledger"); policy != config.APIPolicyPinned {
		t.Fatalf("expected pinned policy, got %q", policy)
	}
	if policy := rt.APIPolicyFor("payments"); policy != config.APIPolicyLatestCompatible {
		t.Fatalf("expected default latest-compatible policy, got %q", policy)
	}
}

func TestNewUsesContextOverride(t *testing.T) {
	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth:     config.Auth{Method: config.AuthMethodNone},
			},
			"cloud": {
				Kind:     config.ContextKindCloud,
				CloudURL: "https://app.formance.cloud/api",
				Auth:     config.Auth{Method: config.AuthMethodCloudDevice},
			},
		},
	})

	rt, err := New(context.Background(), Options{
		ConfigPath:      configPath,
		ContextOverride: config.ContextOverride{Name: "cloud"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rt.ContextName != "cloud" {
		t.Fatalf("expected cloud context, got %q", rt.ContextName)
	}
	if rt.Target.Kind != TargetKindCloud || rt.Target.URL != "https://app.formance.cloud/api" {
		t.Fatalf("unexpected target: %#v", rt.Target)
	}
}

func TestNewRequiresConfigPath(t *testing.T) {
	_, err := New(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected config path error")
	}
}

func TestHTTPClientUsesContextAuth(t *testing.T) {
	ctx := context.Background()
	store := credentials.NewMemoryStore()
	if err := store.Set(ctx, "token-ref", "runtime-token"); err != nil {
		t.Fatalf("set token: %v", err)
	}

	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth: config.Auth{
					Method:   config.AuthMethodToken,
					TokenRef: "token-ref",
				},
			},
		},
	})

	rt, err := New(ctx, Options{ConfigPath: configPath, Credentials: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	client, err := rt.HTTPClient(ctx)
	if err != nil {
		t.Fatalf("runtime http client: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer runtime-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rsp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rsp.StatusCode)
	}
}

func TestCloudClientCredentialsDefaultOrganizationScopes(t *testing.T) {
	rt := &Runtime{
		Context: config.Context{
			Kind: config.ContextKindCloud,
			Auth: config.Auth{
				Method:    config.AuthMethodClientCredentials,
				IssuerURL: "https://app.formance.cloud/api",
				ClientID:  "client",
				SecretRef: "secret-ref",
			},
		},
		Target: Target{Kind: TargetKindCloud},
	}

	authConfig := rt.authForTarget()
	if !containsString(authConfig.Scopes, "organization:ListStacks") {
		t.Fatalf("expected default organization scopes, got %#v", authConfig.Scopes)
	}
}

func TestCloudDeviceDefaultOrganizationScopes(t *testing.T) {
	rt := &Runtime{
		Context: config.Context{
			Kind: config.ContextKindCloud,
			Auth: config.Auth{
				Method:   config.AuthMethodCloudDevice,
				TokenRef: "root-token-ref",
			},
		},
		Target: Target{Kind: TargetKindCloud},
	}

	authConfig := rt.authForTarget()
	if !containsString(authConfig.Scopes, "organization:ListStacks") {
		t.Fatalf("expected default organization scopes, got %#v", authConfig.Scopes)
	}
}

func TestStackClientCredentialsDoNotDefaultOrganizationScopes(t *testing.T) {
	rt := &Runtime{
		Context: config.Context{
			Kind: config.ContextKindStack,
			Auth: config.Auth{
				Method:    config.AuthMethodClientCredentials,
				IssuerURL: "https://auth.example",
				ClientID:  "client",
				SecretRef: "secret-ref",
			},
		},
		Target: Target{Kind: TargetKindStack},
	}

	authConfig := rt.authForTarget()
	if len(authConfig.Scopes) != 0 {
		t.Fatalf("expected no default scopes for stack targets, got %#v", authConfig.Scopes)
	}
}

func TestHTTPVersionsClientParsesVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	versions, err := HTTPVersionsClient{BaseURL: server.URL + "/api"}.GetVersions(context.Background())
	if err != nil {
		t.Fatalf("get versions: %v", err)
	}
	if len(versions) != 1 || versions[0].Product != "ledger" || versions[0].Version != "2.3.4" || !versions[0].Health {
		t.Fatalf("unexpected versions: %#v", versions)
	}
}

func TestResolveAPIVersionFetchesComponentVersion(t *testing.T) {
	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth:     config.Auth{Method: config.AuthMethodNone},
			},
		},
	})
	rt, err := New(context.Background(), Options{
		ConfigPath: configPath,
		VersionsClient: staticVersionsClient{versions: []capabilities.ComponentVersion{
			{Product: "ledger", Version: "2.3.4", Health: true},
		}},
		Compatibility: capabilities.ComponentCompatibility{
			{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []capabilities.APIVersion{"v1", "v2"}},
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	selected, err := rt.ResolveAPIVersion(context.Background(), capabilities.VersionResolutionRequest{
		Product:         "ledger",
		Feature:         "listTransactions",
		HandlerVersions: []capabilities.APIVersion{"v1", "v2", "v3"},
	})
	if err != nil {
		t.Fatalf("resolve api version: %v", err)
	}
	if selected != "v2" {
		t.Fatalf("expected v2, got %q", selected)
	}
}

type staticVersionsClient struct {
	versions []capabilities.ComponentVersion
}

func (c staticVersionsClient) GetVersions(context.Context) ([]capabilities.ComponentVersion, error) {
	return c.versions, nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func writeRuntimeConfig(t *testing.T, cfg config.Config) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.SaveFile(path, cfg); err != nil {
		t.Fatalf("save runtime config: %v", err)
	}
	return path
}
