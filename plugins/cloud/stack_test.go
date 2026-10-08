package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

func stackTestClient(t *testing.T, handler http.HandlerFunc) *httpclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := httpclient.New(server.URL+"/membership", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func stackTestRequest(path string, args ...string) pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{CommandPath: strings.Fields(path), Args: args, Flags: map[string]string{}, ChangedFlags: map[string]bool{}, Context: map[string]string{"organization": "org", "stack": "selected"}}
}

func stackTestJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		t.Error(err)
	}
}

func TestStackMembershipRoutes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		command, method, suffix, query string
		args                           []string
	}{
		{"show", "GET", "/selected", "", nil},
		{"info", "GET", "/selected", "", nil},
		{"version", "GET", "/explicit", "", []string{"explicit"}},
		{"delete", "DELETE", "/selected", "force=false", nil},
		{"disable", "PUT", "/selected/disable", "", nil},
		{"enable", "PUT", "/selected/enable", "", nil},
		{"modules list", "GET", "/selected/modules", "", nil},
		{"modules enable", "POST", "/explicit/modules", "name=ledger", []string{"ledger", "explicit"}},
		{"modules disable", "DELETE", "/selected/modules", "name=ledger", []string{"ledger"}},
		{"users list", "GET", "/selected/users", "", nil},
		{"users unlink", "DELETE", "/selected/users/user%2Fid", "", []string{"user/id"}},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var calls atomic.Int32
			client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				stackAssertRoute(t, r, tc.method, "/membership/organizations/org/stacks"+tc.suffix, tc.query)
				stackTestJSON(t, w, http.StatusOK, `{"data":{"counter":123456789012345678901234567890}}`)
			})
			req := stackTestRequest(tc.command, tc.args...)
			if slicesStackConfirm(tc.command) {
				req.Flags["confirm"] = "true"
			}
			if tc.command == "delete" {
				req.Flags["force"], req.ChangedFlags["force"] = "false", true
			}
			result, err := executeStack(t.Context(), client, req, req.CommandPath)
			if err != nil || string(result) != `{"data":{"counter":123456789012345678901234567890}}` || calls.Load() != 1 {
				t.Fatalf("result=%s calls=%d err=%v", result, calls.Load(), err)
			}
		})
	}
}

func stackAssertRoute(t *testing.T, r *http.Request, method, path, query string) {
	t.Helper()
	if r.Method != method || r.URL.EscapedPath() != path || r.URL.RawQuery != query {
		t.Errorf("wrong route: %s %s", r.Method, r.URL.RequestURI())
	}
}

func slicesStackConfirm(command string) bool {
	return command == "delete" || command == "disable" || command == "enable" || command == "restore" || command == "modules disable" || command == "users unlink"
}

func TestStackDestructiveConfirmationBeforeHTTP(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("unconfirmed command reached Membership") })
	for _, command := range []string{"delete", "disable", "enable", "restore", "upgrade", "modules disable", "users unlink"} {
		args := []string{}
		if strings.Contains(command, " ") {
			args = []string{"resource"}
		}
		req := stackTestRequest(command, args...)
		if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
			t.Errorf("%s lacked confirmation", command)
		}
	}
}

func TestStackRestoreWaitsOnMembershipOnly(t *testing.T) {
	t.Parallel()
	var restores, polls atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/restore") {
			stackAssertRoute(t, r, "PUT", "/membership/organizations/org/stacks/selected/restore", "")
			restores.Add(1)
			stackTestJSON(t, w, 202, `{"data":{"id":"selected","version":"v3.3","status":"PROGRESSING"}}`)
			return
		}
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/selected", "")
		polls.Add(1)
		stackTestJSON(t, w, 200, `{"data":{"id":"selected","version":"v3.3","status":"READY"}}`)
	})
	req := stackTestRequest("restore")
	req.Flags["confirm"] = "true"
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
		t.Fatal(err)
	}
	if restores.Load() != 1 || polls.Load() != 1 {
		t.Fatalf("restores=%d polls=%d", restores.Load(), polls.Load())
	}
}

func TestStackUpgradeSameVersionWaitsForReadyAndVersion(t *testing.T) {
	t.Parallel()
	var reads atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/selected", "")
		status, version := "PROGRESSING", "v4.0"
		switch reads.Add(1) {
		case 2:
			status, version = "READY", "v3.3"
		case 3:
			status = "READY"
		}
		stackTestJSON(t, w, 200, `{"data":{"id":"selected","regionID":"eu","status":"`+status+`","version":"`+version+`"}}`)
	})
	req := stackTestRequest("upgrade")
	req.Flags["confirm"], req.Flags["wait-timeout"] = "true", "5s"
	data, err := executeStack(t.Context(), client, req, req.CommandPath)
	if err != nil || reads.Load() != 3 {
		t.Fatalf("data=%s reads=%d err=%v", data, reads.Load(), err)
	}
	ready, err := stackReadiness(data, "org", "selected", "v4.0")
	if err != nil || !ready {
		t.Fatalf("upgrade returned before readiness: data=%s err=%v", data, err)
	}
}

func TestStackUpgradeSameVersionNoWaitReturnsCurrent(t *testing.T) {
	t.Parallel()
	const current = `{"data":{"id":"selected","regionID":"eu","status":"PROGRESSING","version":"v4.0","counter":9007199254740993}}`
	var reads atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/selected", "")
		reads.Add(1)
		stackTestJSON(t, w, 200, current)
	})
	req := stackTestRequest("upgrade")
	req.Flags["confirm"], req.Flags["no-wait"] = "true", "true"
	data, err := executeStack(t.Context(), client, req, req.CommandPath)
	if err != nil || string(data) != current || reads.Load() != 1 {
		t.Fatalf("data=%s reads=%d err=%v", data, reads.Load(), err)
	}
}

func TestStackUpgradeSameVersionRejectsDisabled(t *testing.T) {
	t.Parallel()
	for _, state := range []string{`"status":"DISABLED"`, `"status":"READY","state":"DISABLED"`} {
		t.Run(state, func(t *testing.T) {
			var reads atomic.Int32
			client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/selected", "")
				reads.Add(1)
				stackTestJSON(t, w, 200, `{"data":{"id":"selected","regionID":"eu","version":"v4.0",`+state+`}}`)
			})
			req := stackTestRequest("upgrade")
			req.Flags["confirm"] = "true"
			data, err := executeStack(t.Context(), client, req, req.CommandPath)
			if err == nil || !strings.Contains(err.Error(), "DISABLED") || len(data) != 0 || reads.Load() != 1 {
				t.Fatalf("data=%s reads=%d err=%v", data, reads.Load(), err)
			}
		})
	}
}

func TestStackUpgradeSameVersionWaitCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var reads atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/selected", "")
		if reads.Add(1) == 2 {
			cancel()
		}
		stackTestJSON(t, w, 200, `{"data":{"id":"selected","regionID":"eu","status":"PROGRESSING","version":"v4.0"}}`)
	})
	req := stackTestRequest("upgrade")
	req.Flags["confirm"] = "true"
	data, err := executeStack(ctx, client, req, req.CommandPath)
	if !errors.Is(err, context.Canceled) || len(data) != 0 || reads.Load() != 2 {
		t.Fatalf("data=%s reads=%d err=%v", data, reads.Load(), err)
	}
}

func TestStackCreateCatalogRejectionNeverCreates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, requested, catalog string }{
		{"different line", "v4.0", `{"data":[{"name":"v3.3"}]}`},
		{"patch is not exact", "v4.0", `{"data":[{"name":"v4.0.0"}]}`},
		{"deprecated", "v4.0", `{"data":[{"name":"v4.0","deprecated":true}]}`},
		{"requested version absent", "v4.1", `{"data":[{"name":"v4.0"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/membership/organizations/org/regions/eu/versions" {
					t.Errorf("catalog rejection sent %s %s", r.Method, r.URL.Path)
				}
				stackTestJSON(t, w, 200, tc.catalog)
			})
			req := stackTestRequest("create", "sandbox")
			req.Flags["region"], req.Flags["version"], req.ChangedFlags["version"] = "eu", tc.requested, true
			if result, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil || len(result) != 0 {
				t.Fatalf("created with rejected catalog: %s %v", result, err)
			}
		})
	}
}

func TestStackCreateNoWaitUsesBodyAndDoesNotPoll(t *testing.T) {
	t.Parallel()
	var creates, reads atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			reads.Add(1)
			if !strings.HasSuffix(r.URL.Path, "/regions/eu/versions") {
				t.Errorf("no-wait polled %s", r.URL.Path)
			}
			stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
		case "POST":
			creates.Add(1)
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body["version"]) != `"v4.0"` || string(body["regionID"]) != `"eu"` || string(body["metadata"]) != `{"purpose":"test"}` {
				t.Errorf("wrong create body: %v", body)
			}
			stackTestJSON(t, w, 202, `{"data":{"id":"created","organizationId":"org","status":"PROGRESSING","count":9007199254740993}}`)
		default:
			t.Errorf("unexpected %s", r.Method)
		}
	})
	req := stackTestRequest("create")
	req.Body = json.RawMessage(`{"name":"sandbox","regionID":"eu","metadata":{"purpose":"test"}}`)
	req.Flags["no-wait"] = "true"
	result, err := executeStack(t.Context(), client, req, []string{"cloud", "stack", "create"})
	if err != nil || !strings.Contains(string(result), "9007199254740993") || creates.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("result=%s creates=%d reads=%d err=%v", result, creates.Load(), reads.Load(), err)
	}
}

func TestStackCreateWaitUsesStableReturnedID(t *testing.T) {
	t.Parallel()
	var creates, polls atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/membership/organizations/org/regions/eu/versions":
			stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
		case "/membership/organizations/org/stacks":
			if r.Method != "POST" {
				t.Error("wait re-listed stacks instead of using created ID")
			}
			creates.Add(1)
			stackTestJSON(t, w, 202, `{"data":{"id":"stable","organizationId":"org","status":"READY","version":"v3.3"}}`)
		case "/membership/organizations/org/stacks/stable":
			poll := polls.Add(1)
			version := "v3.3"
			if poll > 1 {
				version = "v4.0"
			}
			stackTestJSON(t, w, 200, `{"data":{"id":"stable","organizationId":"org","status":"READY","version":"`+version+`","counter":9007199254740993}}`)
		default:
			t.Errorf("outside Membership route: %s", r.URL.Path)
		}
	})
	req := stackTestRequest("create", "same-name")
	req.Flags["region"], req.Flags["wait-timeout"] = "eu", "5s"
	result, err := executeStack(t.Context(), client, req, req.CommandPath)
	if err != nil || !strings.Contains(string(result), `"version":"v4.0"`) || creates.Load() != 1 || polls.Load() != 2 {
		t.Fatalf("result=%s creates=%d polls=%d err=%v", result, creates.Load(), polls.Load(), err)
	}
}

func TestStackCreateWaitCancellationNeverRecreates(t *testing.T) {
	t.Parallel()
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "caller cancellation", true: "wait timeout"}[timeout], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var creates, polls atomic.Int32
			client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/versions"):
					stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
				case r.Method == "POST":
					creates.Add(1)
					stackTestJSON(t, w, 202, `{"data":{"id":"stable"}}`)
				case r.Method == "GET":
					polls.Add(1)
					stackTestJSON(t, w, 200, `{"data":{"id":"stable","status":"PROGRESSING"}}`)
					if !timeout {
						cancel()
					}
				default:
					t.Errorf("unexpected route %s", r.URL)
				}
			})
			req := stackTestRequest("create", "sandbox")
			req.Flags["region"], req.Flags["wait-timeout"] = "eu", "50ms"
			result, err := executeStack(ctx, client, req, req.CommandPath)
			want := context.Canceled
			if timeout {
				want = context.DeadlineExceeded
			}
			stackAssertCanceled(t, result, err, want, creates.Load(), polls.Load())
		})
	}
}

func stackAssertCanceled(t *testing.T, result json.RawMessage, err, want error, creates, polls int32) {
	t.Helper()
	if !errors.Is(err, want) || len(result) != 0 || creates != 1 || polls != 1 || !strings.Contains(err.Error(), "stable") {
		t.Fatalf("result=%s err=%v creates=%d polls=%d", result, err, creates, polls)
	}
}

func TestStackUpdatePreservesMetadata(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			stackTestJSON(t, w, 200, `{"data":{"name":"before","metadata":{"keep":"yes"},"counter":9007199254740993}}`)
			return
		}
		var fields map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			t.Error(err)
		}
		if len(fields) != 2 || string(fields["name"]) != `"after"` || string(fields["metadata"]) != `{"keep":"yes"}` {
			t.Errorf("update lost fields or sent server fields: %v", fields)
		}
		stackTestJSON(t, w, 200, `{"data":{"name":"after"}}`)
	})
	req := stackTestRequest("update")
	req.Flags["name"] = "after"
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
		t.Fatal(err)
	}
}

func TestStackHistoryPaginationAndFilters(t *testing.T) {
	t.Parallel()
	want := url.Values{"stackId": {"selected"}, "pageSize": {"25"}, "action": {"stacks.update"}, "userId": {"SYSTEM"}, "key": {"metadata,env"}, "value": {"a=b"}}
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/membership/organizations/org/logs" || r.URL.RawQuery != want.Encode() {
			t.Errorf("wrong logs query: %s", r.URL)
		}
		stackTestJSON(t, w, 200, `{"data":{"data":[],"next":"opaque","pageSize":25,"hasMore":true}}`)
	})
	req := stackTestRequest("history")
	req.Flags = map[string]string{"page-size": "25", "action": "stacks.update", "user-id": "SYSTEM", "data": "metadata,env=a=b"}
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
		t.Fatal(err)
	}
	req.Flags["cursor"] = "cursor"
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("cursor combined with filters")
	}
}

func TestStackUserPolicyKeepsExactInteger(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Method != "PUT" || !strings.HasSuffix(r.URL.Path, "/stacks/selected/users/u1") || string(body) != `{"policyId":9007199254740993}` {
			t.Errorf("policy request: %s %s %s", r.Method, r.URL, body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := stackTestRequest("users link", "u1")
	req.Flags["policy-id"] = "9007199254740993"
	if result, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil || string(result) != "null" {
		t.Fatalf("result=%s err=%v", result, err)
	}
}

func TestStackUpgradeNoWaitAndAlreadyCurrent(t *testing.T) {
	t.Parallel()
	var upgrades atomic.Int32
	version := "v3.3"
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/versions"):
			stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
		case strings.HasSuffix(r.URL.Path, "/upgrade"):
			if r.Method != "PUT" {
				t.Error("upgrade method")
			}
			upgrades.Add(1)
			version = "v4.0"
			w.WriteHeader(http.StatusAccepted)
		default:
			stackTestJSON(t, w, 200, `{"data":{"id":"selected","regionID":"eu","version":"`+version+`"}}`)
		}
	})
	req := stackTestRequest("upgrade")
	req.Flags["confirm"], req.Flags["no-wait"] = "true", "true"
	for range 2 {
		if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
			t.Fatal(err)
		}
	}
	if upgrades.Load() != 1 {
		t.Fatal("repeated upgrade recreated or upgraded current stack")
	}
}

func TestStackInvalidBodyAndWaitSettingsDoNotWrite(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid request reached HTTP") })
	for _, body := range []string{`[]`, `{"name":null}`, `{"name":"x","unknown":1}`, `{"name":"x","metadata":{"bad":1}}`, `{"name":"x","metadata":{"bad":null}}`} {
		req := stackTestRequest("create")
		req.Body = json.RawMessage(body)
		if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	req := stackTestRequest("create", "x")
	req.Flags["wait-timeout"] = "-1s"
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("accepted negative wait timeout")
	}
}

func TestStackSelectsUniqueHostTargetWithoutMutatingRequest(t *testing.T) {
	t.Parallel()
	req := stackTestRequest("show")
	delete(req.Context, "stack")
	req.Context["stacks"] = `{"org":["one","one"],"other":["two"]}`
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks/one", "")
		stackTestJSON(t, w, 200, `{"data":{"id":"one"}}`)
	})
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
		t.Fatal(err)
	}
	if req.Context["stack"] != "" || len(req.Flags) != 0 {
		t.Fatal("execution mutated caller defaults")
	}
	req.Context["stacks"] = `{"org":["one","two"]}`
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil || !strings.Contains(err.Error(), "one, two") {
		t.Fatalf("ambiguous target: %v", err)
	}
	req.Args = []string{".."}
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("dot segment accepted as a stack ID")
	}
}

func TestStackCreateSelectsUniqueRegion(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/membership/organizations/org/regions":
			stackTestJSON(t, w, 200, `{"data":[{"id":"eu"}]}`)
		case "/membership/organizations/org/regions/eu/versions":
			stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
		case "/membership/organizations/org/stacks":
			stackTestJSON(t, w, 202, `{"data":{"id":"created","version":"v4.0"}}`)
		default:
			t.Errorf("unexpected route %s", r.URL)
		}
	})
	req := stackTestRequest("create")
	req.Flags["name"], req.Flags["no-wait"] = "sandbox", "true"
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil {
		t.Fatal(err)
	}
}

func TestStackCreateAcceptsExplicitCatalogVersions(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"v3.3", "v4.1", "v4.0.0"} {
		t.Run(version, func(t *testing.T) {
			var creates atomic.Int32
			client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					stackTestJSON(t, w, 200, `{"data":[{"name":"`+version+`"}]}`)
					return
				}
				creates.Add(1)
				var body struct {
					Version string `json:"version"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Version != version {
					t.Errorf("explicit version changed: %q %v", body.Version, err)
				}
				stackTestJSON(t, w, 202, `{"data":{"id":"created"}}`)
			})
			req := stackTestRequest("create", "sandbox")
			req.Flags["region"], req.Flags["no-wait"], req.Flags["version"] = "eu", "true", version
			req.ChangedFlags["version"] = true
			if _, err := executeStack(t.Context(), client, req, req.CommandPath); err != nil || creates.Load() != 1 {
				t.Fatalf("explicit catalog version failed: calls=%d %v", creates.Load(), err)
			}
		})
	}
}

func TestStackCreateAmbiguousRegionNeverCreates(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/regions") {
			t.Errorf("ambiguous region caused write: %s %s", r.Method, r.URL)
		}
		stackTestJSON(t, w, 200, `{"data":[{"id":"eu"},{"id":"us"}]}`)
	})
	req := stackTestRequest("create", "sandbox")
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("ambiguous region was selected arbitrarily")
	}
}

func TestStackShowByNamePreservesRawMetadataAndRejectsDuplicates(t *testing.T) {
	t.Parallel()
	var duplicate atomic.Bool
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		stackAssertRoute(t, r, "GET", "/membership/organizations/org/stacks", "")
		body := `{"data":[{"id":"one","name":"sandbox","counter":9007199254740993}]}`
		if duplicate.Load() {
			body = `{"data":[{"id":"one","name":"sandbox"},{"id":"two","name":"sandbox"}]}`
		}
		stackTestJSON(t, w, 200, body)
	})
	req := stackTestRequest("show")
	req.Flags["name"] = "sandbox"
	data, err := executeStack(t.Context(), client, req, req.CommandPath)
	if err != nil || !strings.Contains(string(data), "9007199254740993") {
		t.Fatalf("raw metadata lost: %s %v", data, err)
	}
	duplicate.Store(true)
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("duplicate name selected arbitrarily")
	}
}

func TestStackCreateWaitFailureNeverRetriesWrite(t *testing.T) {
	t.Parallel()
	var creates, polls atomic.Int32
	client := stackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/versions"):
			stackTestJSON(t, w, 200, `{"data":[{"name":"v4.0"}]}`)
		case r.Method == "POST":
			creates.Add(1)
			stackTestJSON(t, w, 202, `{"data":{"id":"stable"}}`)
		default:
			polls.Add(1)
			stackTestJSON(t, w, 503, `{"errorCode":"UNAVAILABLE"}`)
		}
	})
	req := stackTestRequest("create", "sandbox")
	req.Flags["region"] = "eu"
	if data, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil || len(data) != 0 || creates.Load() != 1 || polls.Load() != 1 {
		t.Fatalf("data=%s err=%v creates=%d polls=%d", data, err, creates.Load(), polls.Load())
	}
}

func TestStackBodyConflictsDoNotReachHTTP(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("body conflict caused HTTP") })
	req := stackTestRequest("create", "first")
	req.Body = json.RawMessage(`{"name":"different","regionID":"eu"}`)
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("conflicting name silently overwritten")
	}
	req = stackTestRequest("create", "first")
	req.Flags["data"] = `{"metadata":{"x":"y"}}`
	if _, err := executeStack(t.Context(), client, req, req.CommandPath); err == nil {
		t.Fatal("unread host body ignored")
	}
}

func TestStackWaitRejectsChangedIdentity(t *testing.T) {
	t.Parallel()
	client := stackTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		stackTestJSON(t, w, 200, `{"data":{"id":"different","status":"READY","version":"v4.0"}}`)
	})
	if _, err := waitStack(t.Context(), client, "org", "stable", "v4.0", time.Second); err == nil {
		t.Fatal("poll accepted another resource")
	}
}
