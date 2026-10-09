package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func buildAuthProduct(t *testing.T) string {
	t.Helper()
	binary := os.Getenv("FCTL_TEST_AUTH_PLUGIN_BINARY")
	if binary == "" {
		t.Skip("set FCTL_TEST_AUTH_PLUGIN_BINARY to test the independently built or published Auth executable")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}

	return binary
}

func authProductAPI(t *testing.T, version *atomic.Value, mutations *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/gateway/auth/_info":
			externalInfoResponse(t, w, version)
		case "/gateway/auth/clients":
			if r.Method == http.MethodPost {
				mutations.Add(1)
				w.WriteHeader(http.StatusCreated)
				externalFixtureResponse(t, w, `{"data":{"id":"product-id","name":"product-client"}}`)
			} else {
				externalFixtureResponse(t, w, `{"data":[{"id":"product-id","name":"distributed-client"}]}`)
			}
		default:
			t.Errorf("unexpected Auth product request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func installAuthProduct(t *testing.T, args []string, source, binary, version string) {
	t.Helper()
	if source == "local" {
		runAuthDistribution(t, args, "plugins", "install", "--service", "auth", "--binary", binary)
		return
	}
	registry, _ := authOCIRegistry(t)
	manager, err := pluginmanager.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	release := publishDistributionAuth(t, manager, registry.URL, binary, version, 1)
	catalogue := filepath.Join(t.TempDir(), "catalogue.json")
	writeAuthCatalogue(t, catalogue, release)
	runAuthDistribution(t, args, "plugins", "sync", "--service", "auth", "--catalogue", catalogue)
}

func assertAuthProductExecution(t *testing.T, args []string, version *atomic.Value, mutations *atomic.Int32) {
	t.Helper()
	out := runAuthDistribution(t, args, "auth", "clients", "list")
	if !json.Valid([]byte(out)) || !strings.Contains(out, "distributed-client") {
		t.Fatalf("product response: %s", out)
	}
	out = runAuthDistribution(t, args, "auth", "clients", "create", "--data", `{"name":"product-client"}`)
	if !json.Valid([]byte(out)) || mutations.Load() != 1 {
		t.Fatalf("product mutation: %s writes=%d", out, mutations.Load())
	}
	version.Store("1.0.1")
	out, _, err := executeExternalCLI(t, append(append([]string{}, args...), "auth", "clients", "create", "--data", `{"name":"must-not-write"}`))
	if err == nil || out != "" || mutations.Load() != 1 {
		t.Fatalf("product drift: %s %v writes=%d", out, err, mutations.Load())
	}
}

func assertAuthProductOffline(t *testing.T, args []string) {
	t.Helper()
	out := runAuthDistribution(t, args, "auth", "--help")
	if !strings.Contains(out, "Manage Auth clients") {
		t.Fatalf("product offline help: %s", out)
	}
	out = runAuthDistribution(t, args, "__complete", "auth", "")
	if !strings.Contains(out, "clients") {
		t.Fatalf("product offline completion: %s", out)
	}
}

// Exercise the product-owned executable through both supported sources.
func TestAuthProductExecutableDistribution(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	binary := buildAuthProduct(t)
	instance, err := transport.Open(t.Context(), binary, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := instance.GetManifest(t.Context())
	closeErr := instance.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("inspect Auth product: %v %v", err, closeErr)
	}
	for _, source := range []string{"local", "oci"} {
		t.Run(source, func(t *testing.T) {
			var version atomic.Value
			version.Store(manifest.Version)
			var mutations atomic.Int32
			server := authProductAPI(t, &version, &mutations)
			args := authDistributionArgs(t.TempDir(), server.URL)
			installAuthProduct(t, args, source, binary, manifest.Version)
			assertAuthProductExecution(t, args, &version, &mutations)
			server.Close()
			assertAuthProductOffline(t, args)
		})
	}
}

// Live distribution smoke test, explicitly enabled for a published version.
func TestAuthOfficialCatalogueDiscovery(t *testing.T) {
	serviceVersion := os.Getenv("FCTL_TEST_AUTH_OFFICIAL_VERSION")
	if serviceVersion == "" {
		t.Skip("set FCTL_TEST_AUTH_OFFICIAL_VERSION to verify public catalogue and GHCR discovery")
	}
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	var version atomic.Value
	version.Store(serviceVersion)
	var mutations atomic.Int32
	server := authProductAPI(t, &version, &mutations)
	args := authDistributionArgs(t.TempDir(), server.URL)
	assertAuthProductExecution(t, args, &version, &mutations)
	out := runAuthDistribution(t, args, "plugins", "show", "--service", "auth")
	if !strings.Contains(out, `"version": "`+serviceVersion+`"`) || !strings.Contains(out, `"revision": 1`) {
		t.Fatalf("official Auth lock: %s", out)
	}
	server.Close()
	assertAuthProductOffline(t, args)
}
