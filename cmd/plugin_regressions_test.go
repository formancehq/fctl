package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func TestPluginCloudImplicitTargetInstallsAndUsesCache(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	f := newCLICloudFixture(t)
	var infoRequests atomic.Int32
	addPluginCloudInfoRoute(t, f, &infoRequests)
	dir := t.TempDir()
	f.run(t, dir, "login", "--issuer", f.issuer())
	binary := buildExternalLedger(t, "3.0.0")
	args := []string{"--config-dir", dir, "--profile", "cloud", "--no-browser", "--no-input", "-o", "json"}
	install := append(append([]string{}, args...), "plugins", "install", "--binary", binary)
	out, trace, err := executeExternalCLI(t, install)
	if err != nil {
		t.Fatalf("install with the sole verified Cloud target: %v", err)
	}
	f.assertNoSecrets(t, out+trace)
	manager, err := pluginmanager.New(filepath.Join(dir, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	target := pluginmanager.Target{Profile: "cloud", Organization: "org", Stack: "stack", Endpoint: f.issuer()}
	lock, err := manager.Load(target, "ledger")
	if err != nil || lock.ServiceVersion != "3.0.0" || !lock.Local {
		t.Fatalf("plugin was not locked to the resolved Cloud target: %+v, %v", lock, err)
	}
	if infoRequests.Load() != 1 {
		t.Fatalf("installation did not discover Ledger exactly once: %d", infoRequests.Load())
	}
	help := append(append([]string{}, args...), "ledger", "--help")
	out, trace, err = executeExternalCLI(t, help)
	if err != nil || trace != "" || !strings.Contains(out, "External Ledger pilot") || infoRequests.Load() != 1 {
		t.Fatalf("implicit target help did not use cached external metadata: %q %q %v", out, trace, err)
	}
	list := append(append([]string{}, args...), "ledger", "list")
	for range 2 {
		out, trace, err = executeExternalCLI(t, list)
		if err != nil || trace != "" {
			t.Fatalf("execute the cached plugin with the implicit Cloud target: %q %v", trace, err)
		}
		assertCLICloudJSON(t, out, `{"data":[]}`)
	}
	// Embedded Ledger does not issue /_info for list. These checks prove that
	// fresh roots actually execute the cached external plugin on both invocations.
	if infoRequests.Load() != 3 {
		t.Fatalf("commands silently fell back to embedded Ledger: info requests=%d", infoRequests.Load())
	}
	f.assertCounts(t, 2, 2, 1, 2, 0)
	f.assertTargetSaved(t, dir, "org", "stack")
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if entry.Options.Organization != "" || entry.Options.Stack != "" {
		t.Fatal("implicit plugin selection changed the saved profile defaults")
	}
}

func addPluginCloudInfoRoute(t *testing.T, f *cliCloudFixture, requests *atomic.Int32) {
	t.Helper()
	original := f.server.Config.Handler
	// Install this immutable wrapper before the fixture receives any requests.
	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stack/api/ledger/_info" {
			original.ServeHTTP(w, r)
			return
		}
		requests.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+f.targets[0].stackToken {
			t.Error("Ledger version discovery did not use the resolved target's host credentials")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeCLICloudJSON(t, w, map[string]string{"version": "3.0.0"})
	})
}

func TestPluginBootstrapCancellationReturnsError(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", filepath.Join(t.TempDir(), "unused-catalogue.json"))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started, stopped := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/_info" {
			t.Errorf("unexpected bootstrap request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		close(started)
		cancel()
		<-r.Context().Done()
		close(stopped)
	}))
	defer server.Close()
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL, "--no-input", "ledger", "list"}
	// Cancellation happens while NewRootCommandWithArgs is discovering the
	// version. A panic here is a regression, rather than an expected test result.
	assertCancelledPluginCommand(ctx, t, args)
	select {
	case <-started:
	default:
		t.Fatal("cancellation did not occur during Ledger discovery")
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("canceled bootstrap retained its HTTP request")
	}
}

func TestPluginBootstrapAlreadyCancelledReturnsError(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", filepath.Join(t.TempDir(), "unused-catalogue.json"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", "http://127.0.0.1:1", "--no-input", "ledger", "list"}
	assertCancelledPluginCommand(ctx, t, args)
}

func assertCancelledPluginCommand(ctx context.Context, t *testing.T, args []string) {
	t.Helper()
	root := cmd.NewRootCommandWithArgs(ctx, args)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	if err := root.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled bootstrap returned %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("canceled bootstrap printed output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestPluginMissingBinaryPreservesOfflineMetadata(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	dir := t.TempDir()
	binary := buildExternalLedger(t, "3.0.0")
	args := []string{"--config-dir", dir, "--auth-mode", "none", "--ledger-url", server.URL, "--no-input", "-o", "json"}
	install := append(append([]string{}, args...), "plugins", "install", "--binary", binary, "--service-version", "3.0.0")
	if out, trace, err := executeExternalCLI(t, install); err != nil || !json.Valid([]byte(out)) || trace != "" {
		t.Fatalf("install local plugin for offline metadata checks: %q %q %v", out, trace, err)
	}
	removeCachedPluginBinary(t, dir, server.URL)
	server.Close()
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"help", []string{"ledger", "--help"}, "External Ledger pilot"},
		{"completion", []string{"__complete", "led"}, "ledger\tExternal Ledger pilot"},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, _, err := executeExternalCLI(t, append(append([]string{}, args...), test.args...))
			if err != nil || !strings.Contains(out, test.want) {
				t.Fatalf("missing binary changed cached external metadata: %q %v", out, err)
			}
		})
	}
	out, trace, err := executeExternalCLI(t, append(append([]string{}, args...), "ledger", "list"))
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "verify plugin executable") || out != "" || trace != "" {
		t.Fatalf("missing binary did not fail verification before execution: %q %q %v", out, trace, err)
	}
	if requests.Load() != 0 {
		t.Fatalf("offline metadata or failed verification issued HTTP requests: %d", requests.Load())
	}
}

func removeCachedPluginBinary(t *testing.T, directory, endpoint string) {
	t.Helper()
	manager, err := pluginmanager.New(filepath.Join(directory, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := manager.Load(pluginmanager.Target{Endpoint: endpoint}, "ledger")
	if err != nil {
		t.Fatal(err)
	}
	binary, err := manager.BinaryContext(t.Context(), lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(binary); err != nil {
		t.Fatal(err)
	}
}
