package modules_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/modules"
)

type exchange struct {
	method, path, query, body, key, response string
	status                                   int
}

func fixture(t *testing.T, exchanges []exchange) (*httptest.Server, func()) {
	t.Helper()
	var mu sync.Mutex
	index := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if index >= len(exchanges) {
			t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL)
			w.WriteHeader(500)
			return
		}
		expected := exchanges[index]
		index++
		verifyRequest(t, r, expected)
		status := expected.status
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)
		response := expected.response
		if response == "" && status != 204 {
			response = `{"data":{"id":"result","amount":9007199254740993}}`
		}
		if _, err := io.WriteString(w, response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server, func() {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if index != len(exchanges) {
			t.Errorf("performed %d of %d requests", index, len(exchanges))
		}
	}
}

func assertJSON(t *testing.T, actual, expected []byte) {
	t.Helper()
	decode := func(raw []byte) any {
		t.Helper()
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Errorf("invalid JSON %s: %v", raw, err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(actual), decode(expected)) {
		t.Errorf("JSON: %s; expected %s", actual, expected)
	}
}

type commandCase struct {
	name, command string
	args          []string
	flags         map[string]string
	changed       map[string]bool
	body          string
	exchanges     []exchange
}

func runCase(t *testing.T, tc commandCase) {
	t.Helper()
	server, verify := fixture(t, tc.exchanges)
	defer verify()
	path := strings.Fields(tc.command)
	p := modules.Factories()[path[0]](server.Client())
	var body json.RawMessage
	if tc.body != "" {
		body = json.RawMessage(tc.body)
	}
	response, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: path, Args: tc.args, Flags: tc.flags, ChangedFlags: tc.changed, Body: body, Endpoint: server.URL + "/gateway/service"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(response.Data) {
		t.Errorf("invalid response JSON: %s", response.Data)
	}
}

func TestWebhooksContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	cases := []commandCase{
		{name: "config filters", command: "webhooks list", flags: map[string]string{"config-id": "config/a", "endpoint": "https://example.org/hook"}, exchanges: []exchange{{method: "GET", path: "/configs", query: "id=config%2Fa&endpoint=https%3A%2F%2Fexample.org%2Fhook"}}},
		{name: "create", command: "webhooks create", args: []string{"https://example.org/hook", "ledger.transaction.created", "wallets.created"}, flags: confirm, exchanges: []exchange{{method: "POST", path: "/configs", body: `{"endpoint":"https://example.org/hook","eventTypes":["ledger.transaction.created","wallets.created"],"secret":""}`}}},
		{name: "update omit secret", command: "webhooks update", args: []string{"id/%", "https://example.org/new", "event"}, flags: confirm, exchanges: []exchange{{method: "PUT", path: "/configs/id%2F%25", body: `{"endpoint":"https://example.org/new","eventTypes":["event"]}`}}},
		{name: "update explicit empty secret", command: "webhooks update", args: []string{"id", "https://example.org/new", "event"}, flags: map[string]string{"confirm": "true", "secret": ""}, changed: map[string]bool{"secret": true}, exchanges: []exchange{{method: "PUT", path: "/configs/id", body: `{"endpoint":"https://example.org/new","eventTypes":["event"],"secret":""}`}}},
		{name: "rotate secret", command: "webhooks change-secret", args: []string{"id"}, flags: confirm, exchanges: []exchange{{method: "PUT", path: "/configs/id/secret/change", body: `{"secret":""}`}}},
		{name: "activate", command: "webhooks activate", args: []string{"id"}, flags: confirm, exchanges: []exchange{{method: "PUT", path: "/configs/id/activate"}}},
		{name: "deactivate", command: "webhooks deactivate", args: []string{"id"}, flags: confirm, exchanges: []exchange{{method: "PUT", path: "/configs/id/deactivate"}}},
		{name: "delete", command: "webhooks delete", args: []string{"id"}, flags: confirm, exchanges: []exchange{{method: "DELETE", path: "/configs/id", status: 204}}},
		{name: "send test", command: "webhooks test", args: []string{"id"}, flags: confirm, exchanges: []exchange{{method: "GET", path: "/configs/id/test"}}},
		{name: "delivery filters", command: "webhooks deliveries list", flags: map[string]string{"status": "failed", "config-id": "cfg", "created-at-from": "2026-01-01T00:00:00Z", "created-at-to": "2026-02-01T00:00:00Z", "page-size": "20", "cursor": "opaque +/="}, exchanges: []exchange{{method: "GET", path: "/deliveries", query: "status=failed&configId=cfg&createdAtFrom=2026-01-01T00%3A00%3A00Z&createdAtTo=2026-02-01T00%3A00%3A00Z&pageSize=20&cursor=opaque+%2B%2F%3D"}}},
		{name: "delivery show", command: "webhooks deliveries show", args: []string{"delivery/a"}, exchanges: []exchange{{method: "GET", path: "/deliveries/delivery%2Fa"}}},
		{name: "attempts", command: "webhooks deliveries attempts", args: []string{"delivery"}, flags: map[string]string{"cursor": "next", "page-size": "2"}, exchanges: []exchange{{method: "GET", path: "/deliveries/delivery/attempts", query: "cursor=next&pageSize=2"}}},
		{name: "replay", command: "webhooks deliveries replay", args: []string{"delivery"}, flags: map[string]string{"confirm": "true", "idempotency-key": "replay-1"}, exchanges: []exchange{{method: "POST", path: "/deliveries/delivery/replay", key: "replay-1"}}},
		{name: "replay page", command: "webhooks deliveries replay-bulk", flags: map[string]string{"confirm": "true", "idempotency-key": "bulk-1", "created-at-from": "2026-01-01T00:00:00Z", "created-at-to": "2026-02-01T00:00:00Z", "status": "FAILED,pending", "config-id": "[\"cfg/1\",\"cfg2\"]", "cursor": "next", "page-size": "2"}, exchanges: []exchange{{method: "POST", path: "/deliveries/replay", key: "bulk-1", body: `{"createdAtFrom":"2026-01-01T00:00:00Z","createdAtTo":"2026-02-01T00:00:00Z","statuses":["failed","pending"],"configIds":["cfg/1","cfg2"],"cursor":"next","pageSize":2}`}}},
		{name: "replay defaults", command: "webhooks deliveries replay-bulk", flags: map[string]string{"confirm": "true", "idempotency-key": "bulk-2", "created-at-from": "2026-01-01T00:00:00Z"}, exchanges: []exchange{{method: "POST", path: "/deliveries/replay", key: "bulk-2", body: `{"createdAtFrom":"2026-01-01T00:00:00Z","statuses":["failed","pending"],"pageSize":100}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}

func TestOrchestrationContracts(t *testing.T) {
	t.Parallel()
	cases := []commandCase{
		{name: "workflows list", command: "orchestration workflows list", exchanges: []exchange{{method: "GET", path: "/workflows"}}},
		{name: "workflow show", command: "orchestration workflows show", args: []string{"flow/a"}, exchanges: []exchange{{method: "GET", path: "/workflows/flow%2Fa"}}},
		{name: "workflow create", command: "orchestration workflows create", body: `{"name":"flow","stages":[{"send":{"amount":9007199254740993}}]}`, exchanges: []exchange{{method: "POST", path: "/workflows", body: `{"name":"flow","stages":[{"send":{"amount":9007199254740993}}]}`}}},
		{name: "workflow run", command: "orchestration workflows run", args: []string{"flow"}, flags: map[string]string{"wait": "true", "variable": "user=a=1,asset=USD/2"}, exchanges: []exchange{{method: "POST", path: "/workflows/flow/instances", query: "wait=true", body: `{"user":"a=1","asset":"USD/2"}`}}},
		{name: "workflow run raw", command: "orchestration workflows run", args: []string{"flow"}, body: `{"quantity":9007199254740993}`, exchanges: []exchange{{method: "POST", path: "/workflows/flow/instances", query: "wait=false", body: `{"quantity":9007199254740993}`}}},
		{name: "workflow delete", command: "orchestration workflows delete", args: []string{"flow"}, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "DELETE", path: "/workflows/flow"}}},
		{name: "instances list", command: "orchestration instances list", flags: map[string]string{"workflow": "flow", "running": "true"}, exchanges: []exchange{{method: "GET", path: "/instances", query: "workflowID=flow&running=true"}}},
		{name: "instance show", command: "orchestration instances show", args: []string{"inst"}, exchanges: []exchange{{method: "GET", path: "/instances/inst", response: `{"data":{"id":"inst","workflowID":"flow"}}`}, {method: "GET", path: "/workflows/flow"}}},
		{name: "instance describe", command: "orchestration instances describe", args: []string{"inst"}, exchanges: []exchange{{method: "GET", path: "/instances/inst/history", response: `{"data":[{"stage":{"type":"send"}},{"stage":{"type":"delay"}}]}`}, {method: "GET", path: "/instances/inst/stages/0/history"}}},
		{name: "send event", command: "orchestration instances send-event", args: []string{"inst", "event/a"}, exchanges: []exchange{{method: "POST", path: "/instances/inst/events", body: `{"name":"event/a"}`}}},
		{name: "abort", command: "orchestration instances stop", args: []string{"inst"}, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "PUT", path: "/instances/inst/abort"}}},
		{name: "triggers list", command: "orchestration triggers list", flags: map[string]string{"name": "trigger"}, exchanges: []exchange{{method: "GET", path: "/triggers", query: "name=trigger"}}},
		{name: "trigger show", command: "orchestration triggers show", args: []string{"trigger"}, exchanges: []exchange{{method: "GET", path: "/triggers/trigger"}}},
		{name: "trigger create", command: "orchestration triggers create", args: []string{"event", "flow"}, flags: map[string]string{"name": "trigger", "filter": "data.amount > 0", "vars": "value=data.amount"}, exchanges: []exchange{{method: "POST", path: "/triggers", body: `{"name":"trigger","event":"event","workflowID":"flow","filter":"data.amount > 0","vars":{"value":"data.amount"}}`}}},
		{name: "trigger test", command: "orchestration triggers test", args: []string{"trigger", `{"amount":9007199254740993}`}, exchanges: []exchange{{method: "POST", path: "/v2/triggers/trigger/test", body: `{"amount":9007199254740993}`}}},
		{name: "trigger delete", command: "orchestration triggers delete", args: []string{"trigger"}, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "DELETE", path: "/triggers/trigger"}}},
		{name: "occurrences", command: "orchestration triggers occurrences list", args: []string{"trigger"}, exchanges: []exchange{{method: "GET", path: "/triggers/trigger/occurrences"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}

func TestHTTPFailuresAndNumbers(t *testing.T) {
	t.Parallel()
	for _, status := range []int{400, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()
			failure := `{"errorCode":"VALIDATION","errorMessage":"service rejected request","data":{"amount":9007199254740993}}`
			server, verify := fixture(t, []exchange{{method: "DELETE", path: "/configs/id", status: status, response: failure}})
			defer verify()
			p := modules.Factories()["webhooks"](server.Client())
			response, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"webhooks", "delete"}, Args: []string{"id"}, Flags: map[string]string{"confirm": "true"}, Endpoint: server.URL + "/gateway/service"})
			var httpError *httpclient.Error
			if !errors.As(err, &httpError) || httpError.StatusCode != status || httpError.Code != "VALIDATION" {
				t.Fatalf("HTTP failure: %v", err)
			}
			if string(response.Data) != failure {
				t.Errorf("lost partial response: %s", response.Data)
			}
		})
	}
	server, verify := fixture(t, []exchange{{method: "GET", path: "/configs", response: `{"amount":9007199254740993}`}, {method: "GET", path: "/configs", response: `not json`}})
	defer verify()
	p := modules.Factories()["webhooks"](server.Client())
	r := pluginsdk.ExecuteRequest{CommandPath: []string{"webhooks", "list"}, Endpoint: server.URL + "/gateway/service"}
	response, err := p.Execute(t.Context(), r)
	if err != nil || string(response.Data) != `{"amount":9007199254740993}` {
		t.Fatalf("raw response: %s, %v", response.Data, err)
	}
	if _, err := p.Execute(t.Context(), r); err == nil {
		t.Fatal("accepted malformed service JSON")
	}
}

func verifyRequest(t *testing.T, r *http.Request, expected exchange) {
	t.Helper()
	if r.Method != expected.method || r.URL.EscapedPath() != "/gateway/service"+expected.path {
		t.Errorf("request: %s %s; expected %s /gateway/service%s", r.Method, r.URL.EscapedPath(), expected.method, expected.path)
	}
	query, err := url.ParseQuery(expected.query)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(r.URL.Query(), query) {
		t.Errorf("query: %v; expected %v", r.URL.Query(), query)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if expected.body == "" {
		if len(body) > 0 {
			t.Errorf("unexpected body: %s", body)
		}
	} else {
		assertJSON(t, body, []byte(expected.body))
	}
	verifyHeaders(t, r, expected.key, len(body) > 0)
}

func verifyHeaders(t *testing.T, r *http.Request, key string, hasBody bool) {
	t.Helper()
	if r.Header.Get("Idempotency-Key") != key {
		t.Errorf("idempotency key %q; expected %q", r.Header.Get("Idempotency-Key"), key)
	}
	if r.Header.Get("Accept") != "application/json" {
		t.Error("missing JSON accept header")
	}
	if hasBody && r.Header.Get("Content-Type") != "application/json" {
		t.Error("missing JSON content type")
	}
}
