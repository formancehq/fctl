package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
)

func buildExternalLedger(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	source := fmt.Sprintf(`package main
import (
 "context"
 "net/http"
 "github.com/formancehq/fctl/pkg/pluginsdk"
 "github.com/formancehq/fctl/pkg/pluginsdk/transport"
 "github.com/formancehq/fctl/v4/plugins/ledger"
)
type versioned struct { pluginsdk.Plugin }
func (p versioned) GetManifest(ctx context.Context) (pluginsdk.Manifest,error) {
 m,err:=p.Plugin.GetManifest(ctx)
 m.Version=%q
 m.Root.Short="External Ledger pilot"
 m.Root.Long="External Ledger pilot.\n"+m.Root.Long
 return m,err
}
func main() { transport.Serve(func(c *http.Client) pluginsdk.Plugin { return versioned{ledger.New(c)} }) }
`, version)
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "plugin")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, path) //nolint:gosec // Fixed Go build command and test-owned source/output paths.
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build external Ledger: %v: %s", err, output)
	}
	return binary
}

func executeExternalCLI(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	root := cmd.NewRootCommandWithArgs(t.Context(), args)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(t.Context())
	return stdout.String(), stderr.String(), err
}

func TestExternalLedgerCLI(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	binary := buildExternalLedger(t, "3.0.0")
	for _, mode := range []string{"none", "client-credentials"} {
		t.Run(mode, func(t *testing.T) { testExternalLedgerMode(t, binary, mode) })
	}
}

func testExternalLedgerMode(t *testing.T, binary, mode string) {
	var mutations atomic.Int32
	var version atomic.Value
	version.Store("3.0.0")
	server := externalLedgerAPI(t, mode, &mutations, &version)
	defer server.Close()
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", mode, "--ledger-url", server.URL + "/gateway/ledger", "--no-input", "-o", "json"}
	if mode == "client-credentials" {
		t.Setenv("FCTL_CLIENT_SECRET", "host-secret")
		args = append(args, "--client-id", "test", "--token-url", server.URL+"/token")
	}
	install := append(append([]string{}, args...), "plugins", "install", "--binary", binary, "--service-version", "3.0.0")
	out, trace, err := executeExternalCLI(t, install)
	if err != nil || !strings.Contains(out, `"source": "local"`) {
		t.Fatalf("install: %s %s %v", out, trace, err)
	}
	testExternalLedgerRead(t, args)
	testExternalLedgerBulk(t, args, &mutations, &version)
	server.Close()
	testExternalOfflineMetadata(t, args)
}

func externalLedgerAPI(t *testing.T, mode string, mutations *atomic.Int32, version *atomic.Value) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			externalTokenResponse(t, w, r)
			return
		}
		if mode == "client-credentials" && r.Header.Get("Authorization") != "Bearer host-access-token" {
			t.Error("service request did not use host authentication")
		}
		switch r.URL.Path {
		case "/gateway/ledger/_info":
			externalInfoResponse(t, w, version)
		case "/gateway/ledger/v3/":
			externalFixtureResponse(t, w, `{"data":[{"name":"books","createdAt":"2026-10-09T00:00:00Z"}]}`)
		case "/gateway/ledger/v3/books/bulk":
			mutations.Add(1)
			externalFixtureResponse(t, w, `{"data":[{"responseType":"ERROR","errorCode":"TEST","id":90071992547409930001}]}`)
		default:
			t.Errorf("unexpected external request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func externalTokenResponse(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
	}
	if r.Form.Get("client_secret") != "host-secret" && r.Header.Get("Authorization") == "" {
		t.Error("OAuth2 credentials were not supplied by the host")
	}
	externalFixtureResponse(t, w, `{"access_token":"host-access-token","token_type":"Bearer","expires_in":3600}`)
}
func externalFixtureResponse(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()
	if _, err := fmt.Fprint(w, data); err != nil {
		t.Error(err)
	}
}
func externalInfoResponse(t *testing.T, w http.ResponseWriter, version *atomic.Value) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{"version": version.Load()}); err != nil {
		t.Error(err)
	}
}

func testExternalLedgerRead(t *testing.T, args []string) {
	t.Helper()
	list := append(append([]string{}, args...), "-d", "ledger", "list")
	out, trace, err := executeExternalCLI(t, list)
	if err != nil || !strings.Contains(out, "books") || !strings.Contains(trace, "HTTP") {
		t.Fatalf("external list/debug: %s %s %v", out, trace, err)
	}
	if strings.Contains(trace, "host-secret") || strings.Contains(trace, "host-access-token") {
		t.Fatal("debug trace disclosed host credentials")
	}
}
func testExternalLedgerBulk(t *testing.T, args []string, mutations *atomic.Int32, version *atomic.Value) {
	t.Helper()
	bulk := append(append([]string{}, args...), "ledger", "--ledger", "books", "bulk", "--data", `[{"action":"CREATE_TRANSACTION"}]`)
	out, _, err := executeExternalCLI(t, bulk)
	if err == nil || !strings.Contains(out, "90071992547409930001") || mutations.Load() != 1 || !json.Valid([]byte(out)) {
		t.Fatalf("partial results/exact integer/no retry: %s %v mutations=%d", out, err, mutations.Load())
	}
	version.Store("3.0.1")
	out, _, err = executeExternalCLI(t, bulk)
	if err == nil || !strings.Contains(err.Error(), "service reports 3.0.1") || out != "" || mutations.Load() != 1 {
		t.Fatalf("version drift must precede writes: %q %v mutations=%d", out, err, mutations.Load())
	}
}
func testExternalOfflineMetadata(t *testing.T, args []string) {
	t.Helper()
	help := append(append([]string{}, args...), "ledger", "--help")
	out, trace, err := executeExternalCLI(t, help)
	if err != nil || !strings.Contains(out, "External Ledger pilot") || trace != "" {
		t.Fatalf("offline cached help: %s %s %v", out, trace, err)
	}
	completion := append(append([]string{}, args...), "__complete", "ledger", "")
	out, trace, err = executeExternalCLI(t, completion)
	if err != nil || !strings.Contains(out, "transactions") {
		t.Fatalf("offline cached completion: %s %s %v", out, trace, err)
	}
}
