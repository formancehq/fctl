package cmd_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

// This deliberately small executable tests the public transport and distribution
// lifecycle independently of private product sources. Product integration tests
// can additionally build github.com/formancehq/auth/misc/fctl-plugin.
func buildDistributionAuth(t *testing.T, version string) string {
	t.Helper()
	source := fmt.Sprintf(`package main
import (
 "context"
 "fmt"
 "net/http"
 "github.com/formancehq/fctl/pkg/pluginsdk"
 "github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
 "github.com/formancehq/fctl/pkg/pluginsdk/transport"
)
type plugin struct { client *http.Client }
func (p *plugin) GetManifest(context.Context) (pluginsdk.Manifest,error) {
 return pluginsdk.Manifest{Name:"auth", Service:"auth", Version:%q, ProtocolVersion:pluginsdk.ProtocolVersion,
 Root:pluginsdk.CommandSpec{Use:"auth",Target:"stack",Short:"Distributed Auth fixture",Subcommands:[]pluginsdk.CommandSpec{
 {Use:"clients",Subcommands:[]pluginsdk.CommandSpec{{Use:"list",Runnable:true},{Use:"create",Runnable:true},
 {Use:"secrets",Subcommands:[]pluginsdk.CommandSpec{{Use:"create CLIENT_ID",Runnable:true,Args:pluginsdk.ArgsSpec{Min:1,Max:1},Flags:[]pluginsdk.FlagSpec{{Name:"data",Type:"string",Body:true}}}}},
 }},
 }}},nil
}
func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse,error) {
 client,err:=httpclient.New(req.Endpoint,p.client)
 if err!=nil { return pluginsdk.ExecuteResponse{},err }
 method:=http.MethodGet
 if req.CommandPath[len(req.CommandPath)-1]=="create" { method=http.MethodPost }
 path:="/clients"
 if len(req.CommandPath)>2 && req.CommandPath[2]=="secrets" { path=fmt.Sprintf("/clients/%%s/secrets",req.Args[0]) }
 data,err:=client.Do(ctx,method,path,nil,req.Body,nil)
 return pluginsdk.ExecuteResponse{Data:data},err
}
func main() { transport.Serve(func(client *http.Client) pluginsdk.Plugin { return &plugin{client:client} }) }
`, version)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "fctl-plugin-auth")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, path) //nolint:gosec // Fixed Go build command and test-owned source and output paths.
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Auth fixture: %v: %s", err, output)
	}
	return binary
}

// Cache real executable metadata without contacting the service or changing
// authentication counters in the host's session and debug tests.
func cacheDistributionAuth(t *testing.T, dir string, target pluginmanager.Target) {
	t.Helper()
	binary := buildDistributionAuth(t, "1.0.0")
	instance, err := transport.Open(t.Context(), binary, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := instance.GetManifest(t.Context())
	closeErr := instance.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("inspect Auth fixture: %v %v", err, closeErr)
	}
	manager, err := pluginmanager.New(filepath.Join(dir, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.InstallLocal(t.Context(), binary, target, manifest); err != nil {
		t.Fatal(err)
	}
}

func authDistributionAPI(t *testing.T, version *atomic.Value, mutations *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/gateway/auth/_info":
			externalInfoResponse(t, w, version)
		case "/gateway/auth/clients":
			if r.Method == http.MethodPost {
				mutations.Add(1)
			}
			externalFixtureResponse(t, w, `{"data":[{"name":"distributed-client"}]}`)
		default:
			t.Errorf("unexpected Auth request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
func authDistributionArgs(dir, endpoint string) []string {
	return []string{"--config-dir", dir, "--auth-mode", "none", "--auth-url", endpoint + "/gateway/auth", "--no-input", "-o", "json"}
}
func runAuthDistribution(t *testing.T, args []string, tail ...string) string {
	t.Helper()
	out, trace, err := executeExternalCLI(t, append(append([]string{}, args...), tail...))
	if err != nil {
		t.Fatalf("%v: stdout=%s stderr=%s err=%v", tail, out, trace, err)
	}
	return out
}
func assertAuthOffline(t *testing.T, args []string) {
	t.Helper()
	out := runAuthDistribution(t, args, "plugins", "show", "--service", "auth")
	if !strings.Contains(out, `"service": "auth"`) {
		t.Fatalf("Auth lock: %s", out)
	}
	out = runAuthDistribution(t, args, "auth", "--help")
	if !strings.Contains(out, "Distributed Auth fixture") {
		t.Fatalf("offline manifest: %s", out)
	}
	out = runAuthDistribution(t, args, "__complete", "auth", "")
	if !strings.Contains(out, "clients") {
		t.Fatalf("offline completion: %s", out)
	}
}
func assertAuthDrift(t *testing.T, args []string, mutations *atomic.Int32) {
	t.Helper()
	out, _, err := executeExternalCLI(t, append(append([]string{}, args...), "auth", "clients", "create"))
	if err == nil || out != "" || mutations.Load() != 1 {
		t.Fatalf("version drift reached mutation: %s %v writes=%d", out, err, mutations.Load())
	}
}
func TestAuthLocalDistributionCLI(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	binary := buildDistributionAuth(t, "1.0.0")
	var version atomic.Value
	version.Store("1.0.0")
	var mutations atomic.Int32
	server := authDistributionAPI(t, &version, &mutations)
	args := authDistributionArgs(t.TempDir(), server.URL)
	out := runAuthDistribution(t, args, "plugins", "install", "--service", "auth", "--binary", binary)
	if !strings.Contains(out, `"source": "local"`) {
		t.Fatalf("local lock: %s", out)
	}
	out = runAuthDistribution(t, args, "auth", "clients", "create")
	if !strings.Contains(out, "distributed-client") || mutations.Load() != 1 {
		t.Fatalf("Auth mutation: %s writes=%d", out, mutations.Load())
	}
	version.Store("1.0.1")
	assertAuthDrift(t, args, &mutations)
	server.Close()
	// Offline metadata must survive a missing cached executable: no subprocess
	// can provide the help or completion manifest after removing these bytes.
	manager, err := pluginmanager.New(filepath.Join(args[1], "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	locks, err := manager.List()
	if err != nil || len(locks) != 1 {
		t.Fatalf("local Auth lock: %+v %v", locks, err)
	}
	cached, err := manager.Binary(locks[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cached); err != nil {
		t.Fatal(err)
	}
	assertAuthOffline(t, args)
}

// Real loopback OCI upload/download requests, with checksum validation on upload.
func authOCIRegistry(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var mu sync.Mutex
	objects := map[string][]byte{}
	reads := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		const base = "/v2/formance/auth/"
		if !strings.HasPrefix(r.URL.Path, base) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Location", base+"blobs/uploads/fixture")
			w.WriteHeader(http.StatusAccepted)
		case http.MethodPut:
			storeAuthOCIUpload(t, w, r, objects)
		case http.MethodGet:
			reads.Add(1)
			data, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			externalFixtureResponse(t, w, string(data))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server, reads
}
func publishDistributionAuth(t *testing.T, manager *pluginmanager.Manager, registry, binary, version string, revision int) pluginmanager.Release {
	t.Helper()
	instance, err := transport.Open(t.Context(), binary, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := instance.GetManifest(t.Context())
	closeErr := instance.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("inspect Auth: %v %v", err, closeErr)
	}
	release, err := manager.Publish(t.Context(), registry+"/formance/auth", pluginmanager.Release{Service: "auth", ServiceVersion: version, Revision: revision, Platform: pluginmanager.CurrentPlatform(), Manifest: manifest}, binary)
	if err != nil {
		t.Fatal(err)
	}
	return release
}
func writeAuthCatalogue(t *testing.T, path string, releases ...pluginmanager.Release) {
	t.Helper()
	data, err := json.Marshal(pluginmanager.Catalogue{SchemaVersion: pluginmanager.SchemaVersion, Releases: releases})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAuthOCIDistributionCLI(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	registry, reads := authOCIRegistry(t)
	manager, err := pluginmanager.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	binary := buildDistributionAuth(t, "1.0.0")
	v1 := publishDistributionAuth(t, manager, registry.URL, binary, "1.0.0", 1)
	next := buildDistributionAuth(t, "1.0.1")
	v2 := publishDistributionAuth(t, manager, registry.URL, next, "1.0.1", 1)
	catalogue := filepath.Join(t.TempDir(), "catalogue.json")
	writeAuthCatalogue(t, catalogue, v1, v2)
	var version atomic.Value
	version.Store("1.0.0")
	var mutations atomic.Int32
	server := authDistributionAPI(t, &version, &mutations)
	args := authDistributionArgs(t.TempDir(), server.URL)
	out := runAuthDistribution(t, args, "plugins", "sync", "--service", "auth", "--catalogue", catalogue)
	if !strings.Contains(out, `"service": "auth"`) || reads.Load() != 3 {
		t.Fatalf("OCI lock/download: %s reads=%d", out, reads.Load())
	}
	runAuthDistribution(t, args, "auth", "clients", "create")
	v1.Revision = 2
	writeAuthCatalogue(t, catalogue, v1, v2)
	runAuthDistribution(t, args, "auth", "clients", "list")
	out = runAuthDistribution(t, args, "plugins", "show", "--service", "auth")
	if !strings.Contains(out, `"revision": 1`) || reads.Load() != 3 {
		t.Fatalf("revision retained: %s reads=%d", out, reads.Load())
	}
	// Reuse the Auth lock's catalogue, rather than the Ledger/default source.
	out = runAuthDistribution(t, args, "plugins", "sync", "--service", "auth")
	if !strings.Contains(out, `"revision": 2`) {
		t.Fatalf("revision upgrade: %s", out)
	}
	version.Store("1.0.1")
	runAuthDistribution(t, args, "auth", "clients", "list")
	if reads.Load() != 6 {
		t.Fatalf("exact new version not downloaded: %d", reads.Load())
	}
	version.Store("1.0.2")
	assertAuthDrift(t, args, &mutations)
	registry.Close()
	version.Store("1.0.1")
	runAuthDistribution(t, args, "auth", "clients", "list")
	server.Close()
	assertAuthOffline(t, args)
}

func TestAuthDistributionRejectsUnknownAndMismatchedServices(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	dir := t.TempDir()
	args := authDistributionArgs(dir, "http://127.0.0.1:1")
	for _, command := range []string{"install", "sync", "show"} {
		out, _, err := executeExternalCLI(t, append(append([]string{}, args...), "plugins", command, "--service", "wallets"))
		if err == nil || !strings.Contains(err.Error(), "unsupported external plugin service") || out != "" {
			t.Fatalf("%s accepted unknown service: %s %v", command, out, err)
		}
	}
	binary := buildDistributionAuth(t, "1.0.0")
	for _, tail := range [][]string{
		{"plugins", "install", "--binary", binary, "--service-version", "1.0.0"},
		{"plugins", "install", "--service", "auth", "--binary", binary, "--service-version", "1.0.1"},
	} {
		out, _, err := executeExternalCLI(t, append(append([]string{}, args...), tail...))
		if err == nil || out != "" {
			t.Fatalf("mismatch accepted: %s %v", out, err)
		}
	}
	manager, err := pluginmanager.New(filepath.Join(dir, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	locks, err := manager.List()
	if err != nil || len(locks) != 0 {
		t.Fatalf("rejected installation wrote locks: %+v %v", locks, err)
	}
}

func storeAuthOCIUpload(t *testing.T, w http.ResponseWriter, r *http.Request, objects map[string][]byte) {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	path := r.URL.Path
	digest := filepath.Base(path)
	if strings.Contains(path, "/blobs/uploads/") {
		digest = r.URL.Query().Get("digest")
		path = "/v2/formance/auth/blobs/" + digest
	}
	checksum := sha256.Sum256(data)
	if digest != "sha256:"+hex.EncodeToString(checksum[:]) {
		t.Error("OCI upload checksum mismatch")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	objects[path] = data
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}
