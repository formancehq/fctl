package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func buildExternalLedger(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	// A synthetic public-SDK plugin exercises host transport and rendering without
	// importing the Ledger product implementation.
	source := fmt.Sprintf(`package main
import (
 "context"
 "fmt"
 "net/http"
 "net/url"
 "github.com/formancehq/fctl/pkg/pluginsdk"
 "github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
 "github.com/formancehq/fctl/pkg/pluginsdk/transport"
)
type plugin struct { client *http.Client }
func (p *plugin) GetManifest(context.Context) (pluginsdk.Manifest,error) {
 return pluginsdk.Manifest{Name:"ledger",Service:"ledger",Version:%q,ProtocolVersion:pluginsdk.ProtocolVersion,
 Root:pluginsdk.CommandSpec{Use:"ledger",Target:"stack",Short:"External Ledger pilot",Flags:[]pluginsdk.FlagSpec{{Name:"ledger",Type:"string",Persistent:true}},Subcommands:[]pluginsdk.CommandSpec{
 {Use:"list",Runnable:true},
 {Use:"bulk",Runnable:true,Flags:[]pluginsdk.FlagSpec{{Name:"data",Type:"string",Body:true}}},
 }}},nil
}
func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse,error) {
 client,err:=httpclient.New(req.Endpoint,p.client)
 if err!=nil { return pluginsdk.ExecuteResponse{},err }
 method,path,query:=http.MethodGet,"/v3/",url.Values{"pageSize":{"100"}}
 if req.CommandPath[len(req.CommandPath)-1]=="bulk" { method=http.MethodPost;path=httpclient.Path("v3",req.Flags["ledger"],"bulk");query=nil }
 data,err:=client.Do(ctx,method,path,query,req.Body,nil)
 // Return an explicit fixture error alongside a result to exercise host partial output.
 if method==http.MethodPost && err==nil { err=fmt.Errorf("synthetic partial failure") }
 return pluginsdk.ExecuteResponse{Data:data},err
}
func main() { transport.Serve(func(client *http.Client) pluginsdk.Plugin { return &plugin{client:client} }) }
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
			if r.Method != http.MethodPost {
				t.Errorf("mutation method = %s", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Equal(body, []byte(externalBulkBody)) {
				t.Errorf("host changed request JSON: %s %v", body, err)
			}
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

const externalBulkBody = `[{"action":"CREATE_TRANSACTION","data":{"postings":[{"source":"world","destination":"bank","amount":9007199254740993,"asset":"USD/2"}]}}]`

func testExternalLedgerBulk(t *testing.T, args []string, mutations *atomic.Int32, version *atomic.Value) {
	t.Helper()
	bulk := append(append([]string{}, args...), "ledger", "--ledger", "books", "bulk", "--data", externalBulkBody)
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
	if err != nil || !strings.Contains(out, "bulk") {
		t.Fatalf("offline cached completion: %s %s %v", out, trace, err)
	}
}

// Prepare executable metadata without contacting the service or consuming host
// authentication counters. The service itself still verifies the version at runtime.
func cacheDistributionLedger(t *testing.T, dir string, target pluginmanager.Target) {
	t.Helper()
	manager, err := pluginmanager.New(filepath.Join(dir, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(target, "ledger"); err == nil {
		return
	} else if !errors.Is(err, pluginmanager.ErrNotInstalled) {
		t.Fatal(err)
	}
	binary := buildExternalLedger(t, "3.0.0")
	instance, err := transport.Open(t.Context(), binary, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := instance.GetManifest(t.Context())
	closeErr := instance.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("inspect Ledger fixture: %v %v", err, closeErr)
	}
	if _, err := manager.InstallLocal(t.Context(), binary, target, manifest); err != nil {
		t.Fatal(err)
	}
}
