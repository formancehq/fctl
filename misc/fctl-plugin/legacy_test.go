package legacy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"
)

func TestMain(m *testing.M) {
	if os.Getenv("FCTL_PLUGIN_V4") == "formance-fctl-v4" {
		transport.Serve(New)
		return
	}
	os.Exit(m.Run())
}

func TestBundleMetadataAndServiceBoundaries(t *testing.T) {
	manifest, err := New(nil).GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "legacy" || manifest.Version != Version || len(manifest.Root.Subcommands) != 7 {
		t.Fatalf("unexpected manifest %#v", manifest)
	}
	for _, test := range []struct {
		path    []string
		service string
	}{
		{[]string{"legacy", "ledger", "list"}, "ledger"},
		{[]string{"legacy", "auth", "clients", "list"}, "auth"},
		{[]string{"legacy", "payments", "accounts", "list"}, "payments"},
		{[]string{"legacy", "wallets", "list"}, "wallets"},
	} {
		service, err := pluginsdk.CommandService(manifest, test.path)
		if err != nil || service != test.service {
			t.Fatalf("%v service=%s error=%v", test.path, service, err)
		}
	}
}

func TestServiceMetadataIsOfflineAndIndependent(t *testing.T) {
	t.Parallel()
	for name, factory := range Factories() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := factory(nil)
			manifest, err := p.GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Version != Version || manifest.Service != name || manifest.Root.Target != "stack" {
				t.Fatalf("invalid manifest: %+v", manifest)
			}
			manifest.Root.Subcommands[0].Use = "modified"
			again, err := p.GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if again.Root.Subcommands[0].Use == "modified" {
				t.Fatal("manifest metadata aliases internal state")
			}
		})
	}
}

func TestStandaloneBundleHTTPAndVersionGuard(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	fixture := &ledgerVersionFixture{t: t, version: "2.4.15"}
	server := httptest.NewServer(fixture)
	defer server.Close()
	httpClient := server.Client()
	httpClient.Transport = headerTransport{base: httpClient.Transport}
	client, err := transport.Open(t.Context(), binary, httpClient, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})
	manifest, err := client.GetManifest(t.Context())
	if err != nil || manifest.Name != "legacy" {
		t.Fatalf("manifest: %v", err)
	}
	result, err := client.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"legacy", "ledger", "list"}, Endpoint: server.URL, Context: map[string]string{"stack-version": "v3.2"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Cursor struct {
			Data []struct {
				Name string
				ID   json.Number
			}
		}
	}
	if err = json.Unmarshal(result.Data, &decoded); err != nil || len(decoded.Cursor.Data) != 1 || decoded.Cursor.Data[0].ID.String() != "9007199254740993" {
		t.Fatalf("number preservation: %s %v", result.Data, err)
	}
	fixture.setVersion("3.0.0-beta.10")
	_, err = client.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"legacy", "ledger", "create"}, Args: []string{"forbidden"}, Flags: map[string]string{"confirm": "true"}, Endpoint: server.URL, Context: map[string]string{"stack-version": "v3.2"}})
	if err == nil {
		t.Fatal("version drift allowed legacy mutation")
	}
	if fixture.mutationCount() != 0 {
		t.Fatal("legacy guard sent mutation")
	}
}

type headerTransport struct{ base http.RoundTripper }

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer synthetic-fixture")
	return h.base.RoundTrip(r)
}

func TestBundlePreservesOmittedAuthUpdateFlags(t *testing.T) {
	fixture := &authUpdateFixture{t: t}
	server := httptest.NewServer(fixture)
	defer server.Close()
	_, err := New(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{Endpoint: server.URL, CommandPath: []string{"legacy", "auth", "clients", "update"}, Args: []string{"owned"}, Flags: map[string]string{"description": "changed", "confirm": "true"}})
	if err != nil {
		t.Fatal(err)
	}
	if fixture.updated["description"] != "changed" || fixture.updated["public"] != true || fixture.updated["trusted"] != true || fixture.updated["name"] != "keep" {
		t.Fatalf("omitted options overwritten: %#v", fixture.updated)
	}
}

type ledgerVersionFixture struct {
	t         *testing.T
	mu        sync.Mutex
	version   string
	mutations int
}

func (f *ledgerVersionFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer synthetic-fixture" {
		f.t.Error("HTTP did not use host client")
	}
	if r.URL.Path == "/_info" {
		fixtureJSON(f.t, w, map[string]any{"data": map[string]string{"version": f.version}})
		return
	}
	if r.Method != http.MethodGet {
		f.mutations++
	}
	if r.URL.Path != "/v2" {
		f.t.Errorf("unexpected route %s", r.URL.Path)
	}
	fixtureJSON(f.t, w, json.RawMessage(`{"cursor":{"data":[{"name":"synthetic","id":9007199254740993}],"hasMore":false}}`))
}
func (f *ledgerVersionFixture) setVersion(version string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.version = version
}
func (f *ledgerVersionFixture) mutationCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mutations
}

type authUpdateFixture struct {
	t       *testing.T
	updated map[string]any
}

func (f *authUpdateFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/_info":
		fixtureJSON(f.t, w, json.RawMessage(`{"version":"v2.5.1"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/clients/owned":
		fixtureJSON(f.t, w, json.RawMessage(`{"data":{"id":"owned","name":"keep","description":"old","public":true,"trusted":true,"scopes":["ledger:read"]}}`))
	case r.Method == http.MethodPut && r.URL.Path == "/clients/owned":
		if err := json.NewDecoder(r.Body).Decode(&f.updated); err != nil {
			f.t.Error(err)
		}
		fixtureJSON(f.t, w, map[string]any{"data": f.updated})
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}
}
func fixtureJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}

func TestLegacyRejectsUnverifiedServiceBeforeOperations(t *testing.T) {
	for _, tc := range []struct {
		name, info, line string
		status           int
	}{
		{"invalid info", `{"data":{}}`, "v3.2", 200},
		{"HTTP failure", `{"error":"unavailable"}`, "v3.2", 503},
		{"modern Auth on modern Stack", `{"version":"v2.5.1"}`, "v4.0-beta", 200},
		{"unknown version", `{"version":"dev"}`, "v3.2", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operations := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/_info" {
					operations++
				}
				w.WriteHeader(tc.status)
				fixtureJSON(t, w, json.RawMessage(tc.info))
			}))
			defer server.Close()
			_, err := NewService("auth", server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"auth", "clients", "list"}, Endpoint: server.URL, Context: map[string]string{"stack-version": tc.line}})
			if err == nil || operations != 0 {
				t.Fatalf("unverified service ran operation: err=%v operations=%d", err, operations)
			}
		})
	}
}

func TestLegacyInvalidRequestAndMissingHostClient(t *testing.T) {
	for _, tc := range []struct {
		service string
		path    []string
	}{
		{"unknown", []string{"unknown", "list"}},
		{"ledger", []string{"ledger", "unknown"}},
		{"ledger", []string{"ledger", "list"}},
	} {
		_, err := NewService(tc.service, nil).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: tc.path, Endpoint: "https://example.invalid"})
		if err == nil {
			t.Fatalf("accepted invalid or unbound service %s", tc.service)
		}
	}
	if _, err := New(nil).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"legacy"}}); err == nil {
		t.Fatal("accepted bundle without service")
	}
}
