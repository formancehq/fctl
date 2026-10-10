package factory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

func TestLegacySelectionIsExplicitAndOffline(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	fixture := newLegacySelectionFixture(t)
	root, settings, directory := fixture.root, fixture.settings, fixture.directory
	provider, _, err := prepareProvider(t.Context(), root, settings, []string{"ledger", "list"}, "ledger")
	if err != nil || provider == nil {
		t.Fatalf("prepare legacy: %v", err)
	}
	manifest, err := provider(nil).GetManifest(t.Context())
	if err != nil || manifest.Version != "1.0.0" || manifest.Name != "ledger" {
		t.Fatalf("legacy metadata: %#v %v", manifest, err)
	}
	selection, err := pluginselection.Load(filepath.Join(directory, "plugins"), pluginselection.Target{Endpoint: fixture.server.URL}, "ledger")
	if err != nil || selection.Provider != "legacy" || selection.ServiceVersion != "2.4.15" {
		t.Fatalf("selection %#v %v", selection, err)
	}
	before := fixture.callCount()
	provider, _, err = prepareProvider(t.Context(), root, settings, []string{"ledger", "--help"}, "ledger")
	if err != nil || provider == nil {
		t.Fatal(err)
	}
	after := fixture.callCount()
	fixture.setVersion("3.0.0-beta.10")
	if before != after {
		t.Fatal("help accessed remote service")
	}
	// A changed service family must never retain the historical provider when
	// modern publication fails. The unavailable catalogue is an explicit override.
	t.Setenv("FCTL_PLUGIN_CATALOGUE", fixture.server.URL+"/missing")
	if provider, _, err = prepareProvider(t.Context(), root, settings, []string{"ledger", "list"}, "ledger"); err == nil || provider != nil {
		t.Fatal("missing modern catalogue fell back to legacy")
	}
}

func TestLegacyBundleMetadataHasNoHTTP(t *testing.T) {
	provider := LegacyBundle()
	registryRoot := &cobra.Command{Use: "fctl"}
	registryRoot.SetContext(t.Context())
	settings := &connection.Settings{}
	settings.Bind(registryRoot)
	manifest, err := provider(nil).GetManifest(t.Context())
	if err != nil || manifest.Name != "legacy" || len(manifest.Root.Subcommands) != 7 {
		t.Fatalf("bundle metadata %#v %v", manifest, err)
	}
}

type legacySelectionFixture struct {
	mu        sync.Mutex
	calls     int
	version   string
	server    *httptest.Server
	root      *cobra.Command
	settings  *connection.Settings
	directory string
}

func newLegacySelectionFixture(t *testing.T) *legacySelectionFixture {
	t.Helper()
	f := &legacySelectionFixture{version: "2.4.15", root: &cobra.Command{Use: "fctl"}, settings: &connection.Settings{}, directory: t.TempDir()}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls++
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/_info" {
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"version": f.version}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(f.server.Close)
	f.root.SetContext(t.Context())
	f.settings.Bind(f.root)
	for name, value := range map[string]string{"config-dir": f.directory, "auth-mode": "none", "ledger-url": f.server.URL} {
		if err := f.root.PersistentFlags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func (f *legacySelectionFixture) callCount() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }
func (f *legacySelectionFixture) setVersion(version string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.version = version
}
