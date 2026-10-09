package pluginhost

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

// referencedRegistry serves real digest-addressed OCI objects published through
// the manager's public API. Its binary is deliberately not executable: bootstrap
// and cached metadata must never launch a downloaded plugin.
type referencedRegistry struct {
	server  *httptest.Server
	mu      sync.Mutex
	objects map[string][]byte
	reads   map[string]int
	version string
}

func newReferencedRegistry(t *testing.T) *referencedRegistry {
	t.Helper()
	r := &referencedRegistry{objects: make(map[string][]byte), reads: make(map[string]int), version: "1.0.0"}
	r.server = httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(r.server.Close)
	return r
}

func (r *referencedRegistry) serve(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := req.URL.Path
	switch req.Method {
	case http.MethodPost:
		w.Header().Set("Location", "/v2/formance/auth/blobs/uploads/fixture")
		w.WriteHeader(http.StatusAccepted)
	case http.MethodPut:
		data, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if strings.Contains(path, "/uploads/") {
			path = "/v2/formance/auth/blobs/" + req.URL.Query().Get("digest")
		}
		r.objects[path] = data
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		r.reads[path]++
		data, exists := r.objects[path]
		if path == "/_info" {
			data, exists = []byte(`{"version":"v`+r.version+`"}`), true
		}
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(path, "/manifests/") {
			w.Header().Set("Content-Type", pluginmanager.ImageManifestMediaType)
		}
		if _, err := w.Write(data); err != nil {
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func referenceJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func referenceHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (r *referencedRegistry) populate(t *testing.T, manager *pluginmanager.Manager) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "metadata-only")
	if err := os.WriteFile(binary, []byte("metadata must not launch this binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	var refs []pluginmanager.CatalogueReference
	for _, version := range []string{"1.0.0", "1.0.1"} {
		manifest := loaderManifest("auth")
		manifest.Version = version
		release, err := manager.Publish(t.Context(), r.server.URL+"/formance/auth", pluginmanager.Release{
			Service: "auth", ServiceVersion: version, Revision: 1, Platform: pluginmanager.CurrentPlatform(), Manifest: manifest,
		}, binary)
		if err != nil {
			t.Fatal(err)
		}
		data := referenceJSON(t, pluginmanager.Catalogue{SchemaVersion: 1, Releases: []pluginmanager.Release{release}})
		path := "/auth/" + version + "/catalogue.json"
		r.mu.Lock()
		r.objects[path] = data
		r.mu.Unlock()
		refs = append(refs, pluginmanager.CatalogueReference{ServiceVersion: version, Catalogue: r.server.URL + path, SHA256: referenceHash(data)})
	}
	index := referenceJSON(t, pluginmanager.Catalogue{SchemaVersion: 2, Plugins: map[string]pluginmanager.ProductCatalogues{"auth": {Releases: refs}}})
	r.mu.Lock()
	r.objects["/index.json"] = index
	r.mu.Unlock()
}

func referencedPreparation(t *testing.T) (servicePreparation, *referencedRegistry) {
	t.Helper()
	r := newReferencedRegistry(t)
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "auth-url", r.server.URL)
	root.SetContext(t.Context())
	root.SetErr(io.Discard)
	manager, err := pluginManager(settings, root)
	if err != nil {
		t.Fatal(err)
	}
	r.populate(t, manager)
	// Only the public root index is redirected; service and OCI traffic still
	// passes through real loopback HTTP endpoints.
	transport := http.DefaultTransport
	manager, err = pluginmanager.New(filepath.Join(settings.Directory, "plugins"), &http.Client{Transport: discoveryTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == pluginmanager.DefaultCatalogue {
			clone := req.Clone(req.Context())
			parsed, parseErr := url.Parse(r.server.URL + "/index.json")
			if parseErr != nil {
				return nil, parseErr
			}
			clone.URL = parsed
			req = clone
		}
		return transport.RoundTrip(req)
	})})
	if err != nil {
		t.Fatal(err)
	}
	return servicePreparation{root: root, settings: settings, manager: manager, service: serviceDescriptor{name: "auth", title: "Auth"}}, r
}

func TestReferencedCatalogueAutomaticDiscoveryAndUpgrade(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	prep, registry := referencedPreparation(t)
	plan := pluginBootstrap{commands: []string{"auth", "probe"}}
	first, err := prep.resolve(t.Context(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.ServiceVersion != "1.0.0" || first.Catalogue != pluginmanager.DefaultCatalogue || first.Local {
		t.Fatalf("unexpected discovered lock: %+v", first)
	}
	registry.mu.Lock()
	initialReads, futureReads := registry.reads["/auth/1.0.0/catalogue.json"], registry.reads["/auth/1.0.1/catalogue.json"]
	registry.version = "1.0.1"
	registry.mu.Unlock()
	if initialReads != 1 || futureReads != 0 {
		t.Fatalf("initial/future catalogue reads: %d/%d", initialReads, futureReads)
	}
	upgraded, err := prep.resolve(t.Context(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.ServiceVersion != "1.0.1" || upgraded.Target != first.Target || upgraded.Catalogue != pluginmanager.DefaultCatalogue {
		t.Fatalf("upgrade lost original index or target: %+v", upgraded)
	}
	locks, err := prep.manager.List()
	if err != nil || len(locks) != 1 || locks[0].ServiceVersion != "1.0.1" {
		t.Fatalf("target upgrade: %+v, %v", locks, err)
	}
}

func TestReferencedCatalogueExplicitSyncAndOfflineMetadata(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	prep, registry := referencedPreparation(t)
	lock, err := Sync(prep.root, prep.settings, "auth", registry.server.URL+"/index.json", "1.0.0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Catalogue != registry.server.URL+"/index.json" || lock.ServiceVersion != "1.0.0" {
		t.Fatalf("sync lock: %+v", lock)
	}
	binary, err := prep.manager.Binary(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(binary); err != nil {
		t.Fatal(err)
	}
	registry.server.Close()
	for _, args := range [][]string{{"auth", "--help"}, {"__complete", "auth", ""}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) { testReferencedOfflineMetadata(t, prep, args) })
	}
}

func testReferencedOfflineMetadata(t *testing.T, prep servicePreparation, args []string) {
	t.Helper()
	root := &cobra.Command{Use: "fctl", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().AddFlagSet(prep.root.PersistentFlags())
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(io.Discard)
	factory, err := PrepareService(t.Context(), root, prep.settings, args, "auth")
	if err != nil || factory == nil {
		t.Fatalf("offline factory: %v", err)
	}
	registered := &plugin.Registry{}
	if err := registered.Register(t.Context(), factory(nil), factory); err != nil {
		t.Fatal(err)
	}
	adapter := plugin.NewCommand(registered, func(context.Context, string) (*api.Client, error) {
		t.Fatal("metadata attempted service execution")
		return nil, context.Canceled
	})
	if err := adapter.AddTo(root); err != nil {
		t.Fatal(err)
	}
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "probe") {
		t.Fatalf("cached command missing: %s", output.String())
	}
}

func TestReferencedCatalogueHashFailureDoesNotInstall(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	prep, registry := referencedPreparation(t)
	registry.mu.Lock()
	registry.objects["/auth/1.0.0/catalogue.json"] = append(registry.objects["/auth/1.0.0/catalogue.json"], '\n')
	registry.mu.Unlock()
	_, err := prep.resolve(t.Context(), pluginBootstrap{commands: []string{"auth", "probe"}})
	if err == nil || !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Fatalf("tampered catalogue: %v", err)
	}
	target, err := pluginServiceTarget(prep.settings, prep.root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prep.manager.Load(target, "auth"); !errors.Is(err, pluginmanager.ErrNotInstalled) {
		t.Fatalf("tampered catalogue persisted a lock: %v", err)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for path, count := range registry.reads {
		if strings.HasPrefix(path, "/v2/") && count != 0 {
			t.Fatalf("tampered catalogue downloaded artifact %s", path)
		}
	}
}
