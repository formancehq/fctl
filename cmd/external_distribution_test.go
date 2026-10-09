package cmd_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

// This serves the OCI upload/download protocol over real HTTP. Published layers
// are actual Go executables, which fctl downloads and starts after resolving the
// deployed version. Manager unit tests separately cover registry failure modes.
func externalOCIRegistry(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var mu sync.Mutex
	objects := make(map[string][]byte)
	reads := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Location", "/v2/formance/ledger/blobs/uploads/test")
			w.WriteHeader(http.StatusAccepted)
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			path := r.URL.Path
			if strings.Contains(path, "/blobs/uploads/") {
				path = "/v2/formance/ledger/blobs/" + r.URL.Query().Get("digest")
			}
			objects[path] = data
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			reads.Add(1)
			data, found := objects[r.URL.Path]
			if !found {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			if _, err := w.Write(data); err != nil {
				t.Error(err)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server, reads
}

func publishExternalLedger(t *testing.T, manager *pluginmanager.Manager, registry string, version string) pluginmanager.Release {
	t.Helper()
	binary := buildExternalLedger(t, version)
	instance, err := transport.Open(t.Context(), binary, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := instance.GetManifest(t.Context())
	closeErr := instance.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("inspect published plugin: %v %v", err, closeErr)
	}
	release, err := manager.Publish(t.Context(), registry+"/formance/ledger", pluginmanager.Release{
		Service: "ledger", ServiceVersion: version, Revision: 1, Platform: pluginmanager.CurrentPlatform(), Manifest: manifest,
	}, binary)
	if err != nil {
		t.Fatal(err)
	}
	return release
}

//nolint:gocognit // A single chronological end-to-end lifecycle proves version and revision transitions.
func TestExternalLedgerDistributionCLI(t *testing.T) {
	registry, reads := externalOCIRegistry(t)
	publisher, err := pluginmanager.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	v300 := publishExternalLedger(t, publisher, registry.URL, "3.0.0")
	v301 := publishExternalLedger(t, publisher, registry.URL, "3.0.1")
	catalogue := filepath.Join(t.TempDir(), "catalogue.json")
	writeCatalogue := func(releases ...pluginmanager.Release) {
		t.Helper()
		data, err := json.Marshal(pluginmanager.Catalogue{SchemaVersion: 1, Releases: releases})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(catalogue, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeCatalogue(v300, v301)
	t.Setenv("FCTL_PLUGIN_CATALOGUE", catalogue)
	var version atomic.Value
	version.Store("3.0.0")
	ledger := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var err error
		switch r.URL.Path {
		case "/_info":
			externalInfoResponse(t, w, &version)
		case "/v3/":
			_, err = fmt.Fprint(w, `{"data":[{"name":"books"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
		if err != nil {
			t.Error(err)
		}
	}))
	defer ledger.Close()
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", ledger.URL, "--no-input", "-o", "json"}
	list := append(append([]string{}, args...), "ledger", "list")
	assertList := func() {
		t.Helper()
		out, trace, err := executeExternalCLI(t, list)
		if err != nil || !strings.Contains(out, "books") {
			t.Fatalf("OCI Ledger CLI: %s %s %v", out, trace, err)
		}
	}
	assertList()
	initialReads := reads.Load()
	if initialReads != 3 {
		t.Fatalf("expected OCI manifest/config/executable, got %d reads", initialReads)
	}
	v300.Revision = 2
	writeCatalogue(v300, v301)
	assertList()
	show := append(append([]string{}, args...), "plugins", "show")
	out, _, err := executeExternalCLI(t, show)
	if err != nil || !strings.Contains(out, `"revision": 1`) || reads.Load() != initialReads {
		t.Fatalf("automatic use must retain plugin revision lock: %s %v", out, err)
	}
	syncArgs := append(append([]string{}, args...), "plugins", "sync")
	out, _, err = executeExternalCLI(t, syncArgs)
	if err != nil || !strings.Contains(out, `"revision": 2`) {
		t.Fatalf("explicit sync upgrades plugin revision: %s %v", out, err)
	}
	version.Store("3.0.1")
	assertList()
	if reads.Load() != initialReads+3 {
		t.Fatalf("service version change must fetch matching plugin: reads=%d", reads.Load())
	}
	registry.Close()
	assertList() // Prepared commands need no catalogue/registry downloads.
	version.Store("3.0.2")
	out, _, err = executeExternalCLI(t, list)
	if err == nil || !strings.Contains(err.Error(), "exact service version") || out != "" {
		t.Fatalf("missing exact release must fail: %s %v", out, err)
	}
	ledger.Close()
	args = append(args, "ledger", "--help")
	out, trace, err := executeExternalCLI(t, args)
	if err != nil || trace != "" || !strings.Contains(out, "External Ledger pilot") {
		t.Fatalf("offline downloaded manifest help: %s %s %v", out, trace, err)
	}
}
