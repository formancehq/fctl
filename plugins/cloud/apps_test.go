package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

const appsEnvelope = `{"cursor":{"pageSize":100,"hasMore":false},"data":[{"id":"app","name":"Books","stackId":"stack","version":9007199254740993}]}`
const appsYAML = "name: Books\nvariables:\n  amount: 9007199254740993\n"

type appsRouteCase struct {
	command, method, path, query, body, wantBody, media string
	flags                                               map[string]string
}

func appsRequest(command string, flags map[string]string, body string) pluginsdk.ExecuteRequest {
	flags = maps.Clone(flags)
	if flags == nil {
		flags = map[string]string{}
	}
	flags["experimental"] = "true"
	req := pluginsdk.ExecuteRequest{CommandPath: append([]string{"cloud", "apps"}, strings.Fields(command)...), Flags: flags, Context: map[string]string{"organizations": `["org1","org2"]`}}
	if body != "" {
		req.Body = json.RawMessage(body)
	}
	return req
}
func appsTestClient(t *testing.T, handler http.HandlerFunc) *httpclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := httpclient.New(server.URL+"/deploy/prefix", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return client
}
func appsEqualJSON(t *testing.T, got, want []byte) {
	t.Helper()
	decode := func(raw []byte) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Fatalf("JSON = %s; want %s", got, want)
	}
}

//nolint:gocognit // table-driven protocol tests keep route and payload assertions together.
func TestAppsEveryRunnableRoute(t *testing.T) {
	cases := []appsRouteCase{
		{command: "list", method: "GET", path: "/apps", query: "cursor=a%2Bb&pageSize=25", flags: map[string]string{"cursor": "a+b", "page-size": "25"}},
		{command: "show", method: "GET", path: "/apps/app", flags: map[string]string{"id": "app"}},
		{command: "create", method: "POST", path: "/apps", flags: map[string]string{"name": "Books", "stack-id": "stack"}, wantBody: `{"name":"Books","stackId":"stack"}`},
		{command: "delete", method: "DELETE", path: "/apps/app", query: "wait=false", flags: map[string]string{"id": "app", "confirm": "true"}},
		{command: "bind-manifest", method: "PUT", path: "/apps/app/manifest", flags: map[string]string{"app-id": "app", "manifest-id": "mf"}, wantBody: `{"manifestId":"mf"}`},
		{command: "unbind-manifest", method: "DELETE", path: "/apps/app/manifest", flags: map[string]string{"app-id": "app", "confirm": "true"}},
		{command: "manifests list", method: "GET", path: "/manifests", query: "pageSize=100"},
		{command: "manifests show", method: "GET", path: "/manifests/mf", flags: map[string]string{"id": "mf"}},
		{command: "manifests create", method: "POST", path: "/manifests", query: "name=Books", flags: map[string]string{"name": "Books"}, body: `{"yaml":"name: Books\nvariables:\n  amount: 9007199254740993\n"}`, wantBody: appsYAML, media: "application/yaml"},
		{command: "manifests update", method: "PATCH", path: "/manifests/mf", flags: map[string]string{"id": "mf", "name": "Books"}, wantBody: `{"name":"Books"}`},
		{command: "manifests delete", method: "DELETE", path: "/manifests/mf", flags: map[string]string{"id": "mf", "confirm": "true"}},
		{command: "manifests download", method: "GET", path: "/manifests/mf/versions/latest", flags: map[string]string{"id": "mf"}, media: "application/x-yaml"},
		{command: "manifests versions list", method: "GET", path: "/manifests/mf/versions", query: "pageSize=100", flags: map[string]string{"manifest-id": "mf"}},
		{command: "manifests versions show", method: "GET", path: "/manifests/mf/versions/9007199254740993", flags: map[string]string{"manifest-id": "mf", "version": "9007199254740993"}},
		{command: "manifests versions push", method: "POST", path: "/manifests/mf/versions", flags: map[string]string{"manifest-id": "mf"}, body: `{"stack":{"modules":{"ledger":{"version":"v3"}},"metadata":{"amount":9007199254740993}}}`, wantBody: `{"stack":{"modules":{"ledger":{"version":"v3"}},"metadata":{"amount":9007199254740993}}}`},
		{command: "deployments list", method: "GET", path: "/deployments", query: "appId=app&pageSize=100", flags: map[string]string{"app-id": "app"}},
		{command: "deployments show", method: "GET", path: "/deployments/dep", query: "include=state", flags: map[string]string{"id": "dep", "include-state": "true"}},
		{command: "deployments logs", method: "GET", path: "/deployments/dep/logs", flags: map[string]string{"id": "dep"}},
		{command: "deployments download", method: "GET", path: "/deployments/dep", flags: map[string]string{"id": "dep"}, media: "application/yaml"},
		{command: "deployments create", method: "POST", path: "/deployments", flags: map[string]string{"app-id": "app", "manifest-id": "mf", "manifest-version": "9007199254740993", "wait": "false"}, wantBody: `{"appId":"app","manifestId":"mf","manifestVersion":9007199254740993}`},
		{command: "variables list", method: "GET", path: "/apps/app/variables", query: "pageSize=100", flags: map[string]string{"id": "app"}},
		{command: "variables create", method: "POST", path: "/apps/app/variables", flags: map[string]string{"id": "app", "key": "API_KEY", "value": "synthetic-secret", "description": "Example"}, wantBody: `{"variable":{"key":"API_KEY","value":"synthetic-secret","description":"Example"}}`},
		{command: "variables delete", method: "DELETE", path: "/apps/app/variables/var", flags: map[string]string{"id": "var", "app-id": "app", "confirm": "true"}},
	}
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.command] = true
		t.Run(tc.command, func(t *testing.T) {
			calls := 0
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tc.method || r.URL.EscapedPath() != "/deploy/prefix"+tc.path || r.URL.RawQuery != tc.query {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if tc.wantBody == "" {
					if len(raw) != 0 {
						t.Errorf("unexpected body")
					}
				} else if tc.media == "application/yaml" {
					if string(raw) != tc.wantBody {
						t.Errorf("YAML body = %q", raw)
					}
				} else {
					appsEqualJSON(t, raw, []byte(tc.wantBody))
				}
				if tc.wantBody != "" {
					media := tc.media
					if media == "" {
						media = "application/json"
					}
					if r.Header.Get("Content-Type") != media {
						t.Errorf("content type = %s", r.Header.Get("Content-Type"))
					}
				}
				if strings.HasSuffix(tc.command, "download") {
					if r.Header.Get("Accept") != tc.media {
						t.Errorf("accept = %s", r.Header.Get("Accept"))
					}
					w.Header().Set("Content-Type", tc.media)
					appsTestWrite(t, w, appsYAML)
				} else {
					w.Header().Set("Content-Type", "application/json")
					appsTestWrite(t, w, appsEnvelope)
				}
			})
			req := appsRequest(tc.command, tc.flags, tc.body)
			result, err := executeApps(t.Context(), client, req, req.CommandPath)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Errorf("calls = %d", calls)
			}
			if strings.HasSuffix(tc.command, "download") {
				var out struct {
					YAML string `json:"yaml"`
				}
				if json.Unmarshal(result, &out) != nil || out.YAML != appsYAML {
					t.Fatalf("YAML lost: %s", result)
				}
			} else {
				appsEqualJSON(t, result, []byte(appsEnvelope))
			}
		})
	}
	var walk func(pluginsdk.CommandSpec, []string)
	walk = func(spec pluginsdk.CommandSpec, path []string) {
		path = append(path, pluginsdk.CommandName(spec))
		if spec.Runnable && !covered[strings.Join(path[1:], " ")] {
			t.Errorf("untested runnable command %v", path)
		}
		for _, child := range spec.Subcommands {
			walk(child, slices.Clone(path))
		}
	}
	walk(appsManifest(), nil)
}

func TestAppsManifestHostGuard(t *testing.T) {
	m := pluginsdk.Manifest{Service: "cloud", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{Use: "cloud", Subcommands: []pluginsdk.CommandSpec{appsManifest()}}}
	req := appsRequest("list", nil, "")
	req.Flags = nil
	if _, err := pluginsdk.NormalizeRequest(m, req); err == nil {
		t.Fatal("experimental host guard missing")
	}
	req.Flags = map[string]string{"experimental": "true"}
	normalized, err := pluginsdk.NormalizeRequest(m, req)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Flags["deploy-app-alias"] != "deploy" {
		t.Fatal("alias default missing")
	}
	service, err := pluginsdk.CommandService(m, req.CommandPath)
	if err != nil || service != "cloud-apps" {
		t.Fatalf("service=%s error=%v", service, err)
	}
}
func TestAppsValidationBeforeHTTP(t *testing.T) {
	cases := []struct {
		command, body string
		flags         map[string]string
	}{
		{command: "list", flags: map[string]string{"experimental": "false"}},
		{command: "list", flags: map[string]string{"page-size": "0"}},
		{command: "show", flags: map[string]string{"id": ".."}},
		{command: "show", body: `{"name":"ignored"}`, flags: map[string]string{"id": "app"}},
		{command: "delete", flags: map[string]string{"id": "app"}},
		{command: "unbind-manifest", flags: map[string]string{"app-id": "app"}},
		{command: "manifests delete", flags: map[string]string{"id": "mf"}},
		{command: "variables delete", flags: map[string]string{"id": "var", "app-id": "app"}},
		{command: "create"}, {command: "create", body: `[]`}, {command: "create", body: `{"name":1}`}, {command: "create", body: `{"name":null}`},
		{command: "create", body: `{"name":"Books","unknown":true}`}, {command: "create", body: `{"name":"Books"}`, flags: map[string]string{"name": "Books"}},
		{command: "create", flags: map[string]string{"data": "-"}},
		{command: "bind-manifest", flags: map[string]string{"app-id": "app"}},
		{command: "manifests update", flags: map[string]string{"id": "mf"}},
		{command: "manifests create", flags: map[string]string{"name": "Books"}},
		{command: "manifests create", body: `{"yaml":12}`, flags: map[string]string{"name": "Books"}},
		{command: "manifests create", body: `{"yaml":"x","extra":true}`, flags: map[string]string{"name": "Books"}},
		{command: "manifests create", body: `{"yaml":"x"}`, flags: map[string]string{"name": "Books", "path": "/missing"}},
		{command: "manifests versions push", flags: map[string]string{"manifest-id": "mf", "path": "/missing"}},
		{command: "manifests versions show", flags: map[string]string{"manifest-id": "mf", "version": "0"}},
		{command: "deployments create", body: `{"appId":"app","manifestId":"mf","manifestVersion":1.5}`},
		{command: "deployments create", body: `{"appId":"app","manifestId":"mf","manifestVersion":9223372036854775808}`},
		{command: "deployments create", flags: map[string]string{"manifest-version": "-1"}},
		{command: "deployments create", flags: map[string]string{"wait-timeout": "0s"}},
		{command: "deployments create", flags: map[string]string{"wait-timeout": "25h"}},
		{command: "variables create", flags: map[string]string{"id": "app"}, body: `{"key":"KEY","value":"secret"}`},
		{command: "variables create", flags: map[string]string{"id": "app"}, body: `{"variable":null}`},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d/%s", i, tc.command), func(t *testing.T) {
			calls := 0
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls++ })
			req := appsRequest(tc.command, tc.flags, tc.body)
			if v, ok := tc.flags["experimental"]; ok {
				req.Flags["experimental"] = v
			}
			if _, err := executeApps(t.Context(), client, req, req.CommandPath); err == nil {
				t.Fatal("validation accepted invalid input")
			}
			if calls != 0 {
				t.Fatalf("sent %d requests", calls)
			}
		})
	}
	// Experimental rejection does not even need a valid client or operation.
	if _, err := executeApps(t.Context(), nil, pluginsdk.ExecuteRequest{}, []string{"bogus"}); err == nil || !strings.Contains(err.Error(), "experimental") {
		t.Fatal(err)
	}
}

//nolint:gocognit // table-driven protocol tests keep route and payload assertions together.
func TestAppsYAMLFiles(t *testing.T) {
	for _, cmd := range []string{"manifests create", "manifests versions push"} {
		t.Run(cmd, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.yaml")
			if err := os.WriteFile(path, []byte(appsYAML), 0600); err != nil {
				t.Fatal(err)
			}
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				if string(raw) != appsYAML || r.Header.Get("Content-Type") != "application/yaml" {
					t.Errorf("file encoded incorrectly: %q", raw)
				}
				appsTestWrite(t, w, `{"data":{"manifestId":"mf","version":9007199254740993}}`)
			})
			req := appsRequest(cmd, map[string]string{"name": "Books", "manifest-id": "mf", "path": path}, "")
			if cmd == "manifests create" {
				delete(req.Flags, "manifest-id")
			} else {
				delete(req.Flags, "name")
			}
			if _, err := executeApps(t.Context(), client, req, req.CommandPath); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, data := range [][]byte{{0, 1, 2}, {0xff}, nil, bytes.Repeat([]byte("a"), appsFileLimit+1)} {
		path := filepath.Join(t.TempDir(), "bad.yaml")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := appsReadFile(t.Context(), path); err == nil {
			t.Fatal("invalid text accepted")
		}
	}
	if _, err := appsReadFile(t.Context(), t.TempDir()); err == nil {
		t.Fatal("directory accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := appsReadFile(ctx, "/missing"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

//nolint:gocognit // verifies artifact permissions, cleanup and preservation on rejected media types.
func TestAppsDownloadAtomicPrivateFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(out, []byte("old"), 0644); err != nil { // #nosec G306 -- synthetic world-readable fixture verifies replacement gets private permissions
		t.Fatal(err)
	}
	client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		appsTestWrite(t, w, appsYAML)
	})
	req := appsRequest("deployments download", map[string]string{"id": "dep", "out": out}, "")
	result, err := executeApps(t.Context(), client, req, req.CommandPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out) // #nosec G304 -- reads only the artifact in t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != appsYAML || stat.Mode().Perm() != 0600 || !bytes.Contains(result, []byte(`"bytes":`)) || bytes.Contains(result, []byte("variables")) {
		t.Fatalf("artifact: %s mode %v result %s", raw, stat.Mode(), result)
	}
	entries, err := os.ReadDir(filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatal("temporary artifact leaked")
	}
	for _, media := range []string{"application/json", "text/event-stream"} {
		t.Run(media, func(t *testing.T) {
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", media)
				appsTestWrite(t, w, `{"data":{}}`)
			})
			if _, err := executeApps(t.Context(), client, req, req.CommandPath); err == nil {
				t.Fatal("fabricated download from incompatible response")
			}
			raw, err := os.ReadFile(out) // #nosec G304 -- reads only the artifact in t.TempDir()
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != appsYAML {
				t.Fatal("file changed after download failure")
			}
		})
	}
}

//nolint:gocognit // table-driven protocol tests keep route and payload assertions together.
func TestAppsWaitActualStatusesAndSingleWrite(t *testing.T) {
	for _, status := range []string{"applied", "planned_and_finished", "errored", "pending", ""} {
		t.Run(status, func(t *testing.T) {
			writes, reads := 0, 0
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					writes++
					appsTestWrite(t, w, `{"data":{"id":"dep","status":"pending"}}`)
				} else {
					reads++
					appsTestWrite(t, w, fmt.Sprintf(`{"data":{"id":"dep","status":%q,"counter":9007199254740993}}`, status))
				}
			})
			req := appsRequest("deployments create", map[string]string{"wait-timeout": "50ms"}, `{"appId":"app","manifestId":"mf","manifestVersion":1}`)
			result, err := executeApps(t.Context(), client, req, req.CommandPath)
			success := status == "applied" || status == "planned_and_finished"
			if (err == nil) != success {
				t.Fatalf("status=%s result=%s err=%v", status, result, err)
			}
			if writes != 1 || reads != 1 {
				t.Fatalf("writes=%d reads=%d", writes, reads)
			}
			if !bytes.Contains(result, []byte("9007199254740993")) {
				t.Fatal("last observed metadata lost")
			}
			if status == "pending" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timeout = %v", err)
			}
		})
	}
	t.Run("delete polls destroy deployment", func(t *testing.T) {
		calls := []string{}
		client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.Path)
			if r.Method == "DELETE" {
				appsTestWrite(t, w, `{"destroyDeploymentId":"destroy"}`)
			} else {
				appsTestWrite(t, w, `{"data":{"id":"destroy","status":"applied"}}`)
			}
		})
		req := appsRequest("delete", map[string]string{"id": "app", "wait": "true", "confirm": "true"}, "")
		if _, err := executeApps(t.Context(), client, req, req.CommandPath); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 2 || !strings.HasSuffix(calls[1], "/deployments/destroy") {
			t.Fatalf("calls=%v", calls)
		}
	})
}

//nolint:gocognit // table-driven protocol tests keep route and payload assertions together.
func TestAppsVariablesSensitiveAndJSONBody(t *testing.T) {
	for _, code := range []int{201, 400} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				appsEqualJSON(t, raw, []byte(`{"variable":{"key":"API_KEY","value":"synthetic-secret"}}`))
				w.WriteHeader(code)
				appsTestWrite(t, w, `{"data":{"id":"v","value":"synthetic-secret","revision":9007199254740993},"errorMessage":"synthetic-secret"}`)
			})
			req := appsRequest("variables create", map[string]string{"id": "app"}, `{"variable":{"key":"API_KEY","value":"synthetic-secret"}}`)
			result, err := executeApps(t.Context(), client, req, req.CommandPath)
			if code == 201 {
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(result, []byte("9007199254740993")) {
					t.Fatal("number rounded")
				}
			} else if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("sensitive diagnostic escaped")
			}
			if calls != 1 {
				t.Fatal("retried a write")
			}
			if bytes.Contains(result, []byte(`synthetic-secret`)) {
				t.Fatal("value leaked")
			}
		})
	}
}
func TestAppsLogsRejectSSEWithoutFabrication(t *testing.T) {
	client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		appsTestWrite(t, w, "data: synthetic log\n\n")
	})
	req := appsRequest("deployments logs", map[string]string{"id": "dep"}, "")
	if data, err := executeApps(t.Context(), client, req, req.CommandPath); err == nil || data != nil {
		t.Fatal("fabricated JSON from SSE")
	}
}
func TestAppsImportsArePublic(t *testing.T) {
	for _, name := range []string{"apps.go", "apps_manifest.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "github.com/") && !strings.HasPrefix(path, "github.com/formancehq/fctl/v4/pkg/pluginsdk") {
				t.Fatalf("private or third-party dependency %s", path)
			}
		}
	}
}

func appsTestWrite(t *testing.T, w io.Writer, value string) {
	t.Helper()
	if _, err := io.WriteString(w, value); err != nil {
		t.Error(err)
	}
}

func TestAppsWritesAndPollingFailuresRetainEvidence(t *testing.T) {
	t.Run("write failure is not retried", func(t *testing.T) {
		calls := 0
		client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusConflict)
			appsTestWrite(t, w, `{"errorCode":"CONFLICT","errorMessage":"Already exists"}`)
		})
		req := appsRequest("create", nil, `{"name":"Books"}`)
		_, err := executeApps(t.Context(), client, req, req.CommandPath)
		var serviceErr *httpclient.Error
		if !errors.As(err, &serviceErr) || serviceErr.StatusCode != http.StatusConflict || calls != 1 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	})
	t.Run("poll failure retains accepted deployment", func(t *testing.T) {
		client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				appsTestWrite(t, w, `{"data":{"id":"dep","status":"pending"}}`)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		})
		req := appsRequest("deployments create", nil, `{"appId":"app","manifestId":"mf","manifestVersion":1}`)
		result, err := executeApps(t.Context(), client, req, req.CommandPath)
		if err == nil {
			t.Fatal("poll failure hidden")
		}
		appsEqualJSON(t, result, []byte(`{"data":{"id":"dep","status":"pending"}}`))
	})
	t.Run("errored response without wait", func(t *testing.T) {
		client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			appsTestWrite(t, w, `{"data":{"id":"dep","status":"errored"}}`)
		})
		req := appsRequest("deployments create", map[string]string{"wait": "false"}, `{"appId":"app","manifestId":"mf","manifestVersion":1}`)
		result, err := executeApps(t.Context(), client, req, req.CommandPath)
		if err == nil || !bytes.Contains(result, []byte("errored")) {
			t.Fatalf("result=%s err=%v", result, err)
		}
	})
}

func TestAppsFreshManifestAndCallerIsolation(t *testing.T) {
	spec := appsManifest()
	spec.Flags[0].RequireTrue = false
	spec.Subcommands[0].Use = "changed"
	fresh := appsManifest()
	if !fresh.Flags[0].RequireTrue || fresh.Subcommands[0].Use != "list" {
		t.Fatal("manifest shares mutable state")
	}
	req := appsRequest("list", nil, "")
	originalFlags := maps.Clone(req.Flags)
	originalContext := maps.Clone(req.Context)
	client := appsTestClient(t, func(w http.ResponseWriter, r *http.Request) { appsTestWrite(t, w, appsEnvelope) })
	if _, err := executeApps(t.Context(), client, req, []string{"list"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(req.Flags, originalFlags) || !reflect.DeepEqual(req.Context, originalContext) {
		t.Fatal("caller state mutated")
	}
}
