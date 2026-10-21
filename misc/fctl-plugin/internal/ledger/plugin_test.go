package ledger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

//nolint:gocognit // Keep the complete HTTP contract fixture and assertions together for review.
func TestHistoricalHTTPContracts(t *testing.T) {
	tests := []struct {
		command, method, path string
		args                  []string
		flags                 map[string]string
		body, payload         string
		query                 url.Values
	}{
		{command: "create", method: "POST", path: "/v2/new", args: []string{"new"}, flags: map[string]string{"bucket": "bucket", "features": "HASH_LOGS=SYNC", "metadata": `["name=hello, world","key=a=b"]`}, payload: `{"bucket":"bucket","features":{"HASH_LOGS":"SYNC"},"metadata":{"name":"hello, world","key":"a=b"}}`},
		{command: "create", method: "POST", path: "/v2/empty", args: []string{"empty"}, payload: `{"bucket":""}`},
		{command: "send", method: "POST", path: "/default/transactions", args: []string{"users:1", huge, "USD/2"}, flags: map[string]string{"metadata": "owner=alice", "reference": "ref"}, payload: `{"postings":[{"source":"world","destination":"users:1","amount":` + huge + `,"asset":"USD/2"}],"metadata":{"owner":"alice"},"reference":"ref"}`},
		{command: "send", method: "POST", path: "/default/transactions", args: []string{"world:fund", "users:1", "0", "USD"}, payload: `{"postings":[{"source":"world:fund","destination":"users:1","amount":0,"asset":"USD"}],"reference":""}`},
		{command: "stats", method: "GET", path: "/default/stats"},
		{command: "server-infos", method: "GET", path: "/_info"},
		{command: "list", method: "GET", path: "/v2", payload: "null", query: url.Values{"includeDeleted": {"false"}}},
		{command: "set-metadata", method: "PUT", path: "/v2/named/metadata", args: []string{"named", "x=1", "x=2", "empty=", "value=a=b"}, payload: `{"x":"2","empty":"","value":"a=b"}`},
		{command: "delete-metadata", method: "DELETE", path: "/v2/named/metadata/key", args: []string{"named", "key"}},
		{command: "accounts list", method: "GET", path: "/v2/default/accounts", flags: map[string]string{"metadata": "owner=alice"}, payload: `{"$and":[{"$match":{"metadata[owner]":"alice"}}]}`},
		{command: "accounts list", method: "GET", path: "/v2/default/accounts", payload: `{"$and":[]}`},
		{command: "accounts show", method: "GET", path: "/default/accounts/users:1", args: []string{"users:1"}},
		{command: "accounts set-metadata", method: "POST", path: "/default/accounts/users:1/metadata", args: []string{"users:1", "key=value"}, payload: `{"key":"value"}`},
		{command: "accounts delete-metadata", method: "DELETE", path: "/v2/default/accounts/users:1/metadata/key", args: []string{"users:1", "key"}},
		{command: "transactions list", method: "GET", path: "/default/transactions", flags: map[string]string{"account": "users:1", "src": "world", "dst": "users:1", "reference": "ref", "start": "2026-10-01T12:30:00.123Z", "end": "2026-10-02T12:30:00Z", "metadata": "owner=alice"}, query: url.Values{"pageSize": {"5"}, "account": {"users:1"}, "source": {"world"}, "destination": {"users:1"}, "reference": {"ref"}, "startTime": {"2026-10-01T12:30:00.123Z"}, "endTime": {"2026-10-02T12:30:00Z"}, "metadata[owner]": {"alice"}}},
		{command: "transactions show", method: "GET", path: "/default/transactions/" + huge, args: []string{huge}},
		{command: "transactions num", method: "POST", path: "/default/transactions", body: `"send [USD/2 10] ( source = @world destination = @users:1 )"`, payload: `{"script":{"plain":"send [USD/2 10] ( source = @world destination = @users:1 )"},"reference":""}`},
		{command: "transactions revert", method: "POST", path: "/default/transactions/12/revert", args: []string{"12"}, query: url.Values{"disableChecks": {"false"}}},
		{command: "transactions revert", method: "POST", path: "/v2/default/transactions/12/revert", args: []string{"12"}, flags: map[string]string{"at-effective-date": "true", "force": "true"}, query: url.Values{"atEffectiveDate": {"true"}, "force": {"true"}}},
		{command: "transactions set-metadata", method: "POST", path: "/default/transactions/12/metadata", args: []string{"12", "key=value"}, payload: `{"key":"value"}`},
		{command: "transactions delete-metadata", method: "DELETE", path: "/v2/default/transactions/12/metadata/key", args: []string{"12", "key"}},
		{command: "schemas list", method: "GET", path: "/v2/default/schemas", query: url.Values{"pageSize": {"15"}, "sort": {"created_at"}, "order": {"desc"}}},
		{command: "schemas get", method: "GET", path: "/v2/default/schemas/1", args: []string{"1"}, flags: map[string]string{"format": "yaml"}},
		{command: "schemas insert", method: "POST", path: "/v2/default/schemas/1", args: []string{"1", "host-read.yaml"}, body: `{"chart":{"users":{"metadata":{"large":` + huge + `}}},"extension":{"n":` + huge + `}}`, payload: `{"chart":{"users":{"metadata":{"large":` + huge + `}}},"extension":{"n":` + huge + `}}`},
		{command: "volumes list", method: "GET", path: "/v2/default/volumes", flags: map[string]string{"metadata": "owner=alice", "address": "users:", "group-by": "2", "insertion-date": "true", "start-time": "2026-10-01T12:00:00Z", "end-time": "2026-10-02T12:00:00Z"}, payload: `{"$and":[{"$match":{"metadata[owner]":"alice"}},{"$match":{"account":"users:"}}]}`, query: url.Values{"pageSize": {"10"}, "groupBy": {"2"}, "insertionDate": {"true"}, "startTime": {"2026-10-01T12:00:00Z"}, "endTime": {"2026-10-02T12:00:00Z"}}},
	}
	for _, test := range tests {
		t.Run(test.command+test.path, func(t *testing.T) {
			var calls atomic.Int32
			wantData := `{"data":{"id":` + huge + `,"amount":` + huge + `},"extra":{"future":true}}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != test.method || r.URL.EscapedPath() != "/gateway/api/ledger"+test.path {
					t.Errorf("got %s %s, want %s %s", r.Method, r.URL.EscapedPath(), test.method, test.path)
				}
				wantQuery := test.query
				if wantQuery == nil {
					wantQuery = url.Values{}
				}
				if !reflect.DeepEqual(r.URL.Query(), wantQuery) {
					t.Errorf("query %v, want %v", r.URL.Query(), wantQuery)
				}
				payload, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if test.payload == "" {
					if len(payload) > 0 {
						t.Errorf("unexpected body: %s", payload)
					}
				} else {
					assertJSON(t, payload, []byte(test.payload))
					if r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("content type %q", r.Header.Get("Content-Type"))
					}
				}
				if r.Header.Get("Accept") != "application/json" {
					t.Errorf("Accept %q", r.Header.Get("Accept"))
				}
				writeJSON(t, w, wantData)
			}))
			t.Cleanup(server.Close)
			r := executeRequest(test.command, test.args, test.flags, test.body)
			r.Endpoint = server.URL + "/gateway/api/ledger"
			result, err := New(server.Client()).Execute(t.Context(), r)
			if err != nil {
				t.Fatal(err)
			}
			if string(result.Data) != wantData || calls.Load() != 1 {
				t.Fatalf("data %s; calls %d", result.Data, calls.Load())
			}
		})
	}
}

func TestValidationBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	tests := []struct {
		command string
		args    []string
		flags   map[string]string
		body    string
	}{
		{"missing", nil, nil, ""}, {"accounts", nil, nil, ""}, {"send", []string{"a", "10"}, nil, ""},
		{"send", []string{"a", "ten", "USD"}, nil, ""}, {"send", []string{"a", "-1", "USD"}, nil, ""},
		{"send", []string{"", "1", "USD"}, nil, ""}, {"send", []string{"a", "1", ""}, nil, ""},
		{"send", []string{"a", "1", "USD"}, map[string]string{"confirm": "false"}, ""},
		{"send", []string{"a", "1", "USD"}, map[string]string{"metadata": "bad"}, ""},
		{"create", []string{".."}, nil, ""}, {"create", []string{"x"}, map[string]string{"features": "bad"}, ""},
		{"create", []string{"x"}, map[string]string{"metadata": `[1]`}, ""},
		{"stats", nil, map[string]string{"ledger": "."}, ""}, {"stats", nil, map[string]string{"unknown": "value"}, ""},
		{"stats", nil, nil, `{}`}, {"accounts show", []string{""}, nil, ""},
		{"accounts list", nil, map[string]string{"metadata": "bad"}, ""},
		{"set-metadata", []string{"x", "bad"}, nil, ""}, {"delete-metadata", []string{"x", ".."}, nil, ""},
		{"transactions show", []string{"abc"}, nil, ""}, {"transactions show", []string{"-1"}, nil, ""},
		{"transactions show", []string{"lastwat"}, nil, ""}, {"transactions show", []string{"last-1"}, nil, ""},
		{"transactions set-metadata", []string{"last", "bad"}, nil, ""},
		{"transactions delete-metadata", []string{"last", "."}, nil, ""},
		{"transactions revert", []string{"12"}, map[string]string{"force": "yes"}, ""},
		{"transactions list", nil, map[string]string{"page-size": "-1"}, ""},
		{"transactions list", nil, map[string]string{"page-size": "0"}, ""},
		{"schemas list", nil, map[string]string{"page-size": "0"}, ""},
		{"volumes list", nil, map[string]string{"page-size": "0"}, ""},
		{"transactions list", nil, map[string]string{"start": "yesterday"}, ""},
		{"transactions list", nil, map[string]string{"end": "yesterday"}, ""},
		{"transactions num", nil, nil, ""}, {"transactions num", nil, nil, `null`},
		{"transactions num", nil, nil, `{"script":{"plain":""}}`},
		{"transactions num", nil, map[string]string{"amount-var": "n=no/USD"}, `"script"`},
		{"transactions num", nil, map[string]string{"amount-var": "n=1"}, `"script"`},
		{"transactions num", nil, map[string]string{"account-var": "bad"}, `"script"`},
		{"transactions num", nil, map[string]string{"timestamp": "today"}, `"script"`},
		{"transactions num", nil, nil, `{"script":{"plain":"script","vars":[]}}`},
		{"transactions num", nil, nil, `{"script":{"plain":"script"},"postings":[]}`},
		{"transactions num", nil, nil, `{"script":{"plain":"script"},"metadata":[]}`},
		{"transactions num", nil, nil, `{"script":{"plain":"script"},"reference":3}`},
		{"schemas get", []string{".."}, nil, ""}, {"schemas get", []string{"1"}, map[string]string{"format": "toml"}, ""},
		{"schemas insert", []string{"1"}, nil, `{}`}, {"schemas insert", []string{"1"}, nil, `{"chart":[]}`},
		{"schemas insert", []string{"1"}, nil, `{"chart":{"a":null}}`},
		{"schemas insert", []string{"1"}, nil, `{"chart":{},"queries":[]}`},
		{"volumes list", nil, map[string]string{"group-by": "-1"}, ""},
		{"volumes list", nil, map[string]string{"start-time": "today"}, ""},
		{"import", []string{"x"}, nil, ""}, {"import", []string{"x"}, nil, `null`},
		{"import", []string{"x"}, nil, `[{"id":1},{"id":-2}]`},
		{"import", []string{"x"}, map[string]string{"resume-from-last-log": "true"}, `[{"id":1},{}]`},
		{"import", []string{"x"}, nil, `"{bad}\n"`},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test), func(t *testing.T) {
			r := executeRequest(test.command, test.args, test.flags, test.body)
			r.Endpoint = server.URL
			if _, err := New(server.Client()).Execute(t.Context(), r); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("validation made %d HTTP calls", calls.Load())
	}
	if _, err := New(nil).Execute(t.Context(), executeRequest("stats", nil, nil, "")); err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatalf("missing-client error: %v", err)
	}
}

func TestCallerCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeJSON(t, w, `{}`) }))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := executeRequest("stats", nil, nil, "")
	r.Endpoint = server.URL
	if _, err := New(server.Client()).Execute(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
