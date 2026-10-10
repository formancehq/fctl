package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/auth"
)

func request(path string, args ...string) pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{CommandPath: strings.Fields(path), Args: args}
}

func approve(req pluginsdk.ExecuteRequest) pluginsdk.ExecuteRequest {
	req.Flags = maps.Clone(req.Flags)
	if req.Flags == nil {
		req.Flags = make(map[string]string)
	}
	req.Flags["confirm"] = "true"
	return req
}

func object(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil || result == nil {
		t.Fatalf("expected JSON object, got %s: %v", data, err)
	}
	return result
}

func assertJSON(t *testing.T, actual []byte, expected string) {
	t.Helper()
	if !json.Valid(actual) || !json.Valid([]byte(expected)) {
		t.Fatalf("expected valid JSON values, got %s and %s", actual, expected)
	}
	var got, want any
	actualDecoder := json.NewDecoder(bytes.NewReader(actual))
	actualDecoder.UseNumber()
	if err := actualDecoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	expectedDecoder := json.NewDecoder(strings.NewReader(expected))
	expectedDecoder.UseNumber()
	if err := expectedDecoder.Decode(&want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON = %s, want %s", actual, expected)
	}
}

func fixture(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestHistoricalRoutesAndRawResults(t *testing.T) {
	// The historical aggregate SDK prefixes these routes with /api/auth. The
	// host now supplies that service prefix through ExecuteRequest.Endpoint.
	const rawResponse = `{"data":{"quantity":9007199254740993,"metadata":{"identifier":"9007199254740993"}}}`
	for _, tc := range []struct {
		path, method, route, body string
		args                      []string
	}{
		{path: "auth clients list", method: "GET", route: "/clients"},
		{path: "auth clients show", method: "GET", route: "/clients/client%2F100%25", args: []string{"client/100%"}},
		{path: "auth clients create", method: "POST", route: "/clients", args: []string{"Legacy App"}, body: `{"name":"Legacy App","public":false,"trusted":false}`},
		{path: "auth clients delete", method: "DELETE", route: "/clients/client%2F100%25", args: []string{"client/100%"}},
		{path: "auth clients secrets create", method: "POST", route: "/clients/client%2F100%25/secrets", args: []string{"client/100%", "production"}, body: `{"name":"production"}`},
		{path: "auth clients secrets delete", method: "DELETE", route: "/clients/client%2F100%25/secrets/secret%2F100%25", args: []string{"client/100%", "secret/100%"}},
		{path: "auth users list", method: "GET", route: "/users"},
		{path: "auth users show", method: "GET", route: "/users/user%2F100%25", args: []string{"user/100%"}},
		{path: "auth clients users list", method: "GET", route: "/users"},
		{path: "auth clients users show", method: "GET", route: "/users/user%2F100%25", args: []string{"user/100%"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			server, calls := historicalRouteFixture(t, tc.method, tc.route, tc.body, rawResponse)
			req := request(tc.path, tc.args...)
			if tc.method != "GET" {
				req = approve(req)
			}
			req.Endpoint = server.URL + "/gateway/api/auth/"
			response, err := auth.New(server.Client()).Execute(t.Context(), req)
			if err != nil {
				t.Fatal(err)
			}
			if string(response.Data) != rawResponse || calls.Load() != 1 {
				t.Fatalf("result = %s; calls = %d", response.Data, calls.Load())
			}
		})
	}
}

func TestCreateInputs(t *testing.T) {
	for _, tc := range []struct {
		name, path, body, want string
		args                   []string
		flags                  map[string]string
	}{
		{name: "legacy flags", path: "auth clients create", args: []string{"app"}, flags: map[string]string{
			"public": "true", "trusted": "true", "description": "false", "redirect-uri": `"https://example.com/callback?a=1,2",https://second.example/callback`, "post-logout-redirect-uri": "https://example.com/logout", "client-scopes": "auth:read,ledger:read",
		}, want: `{"name":"app","description":"false","public":true,"trusted":true,"redirectUris":["https://example.com/callback?a=1,2","https://second.example/callback"],"postLogoutRedirectUris":["https://example.com/logout"],"scopes":["auth:read","ledger:read"]}`},
		{name: "form name flag", path: "auth clients create", flags: map[string]string{"name": "form-app"}, want: `{"name":"form-app","public":false,"trusted":false}`},
		{name: "host body", path: "auth clients create", body: `{"name":"json-app","public":true,"redirectUris":[],"metadata":{"id":"9007199254740993"}}`, flags: map[string]string{"data": "@body.json"}, want: `{"name":"json-app","public":true,"trusted":false,"redirectUris":[],"metadata":{"id":"9007199254740993"}}`},
		{name: "secret form", path: "auth clients secrets create", args: []string{"client"}, flags: map[string]string{"name": "form-secret"}, want: `{"name":"form-secret"}`},
		{name: "secret body", path: "auth clients secrets create", args: []string{"client"}, body: `{"name":"json-secret","metadata":{"owner":"test"}}`, want: `{"name":"json-secret","metadata":{"owner":"test"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := fixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.RawQuery != "" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				assertJSON(t, readRequestBody(t, r), tc.want)
				writeResponse(t, w, `{"data":{"id":"created"}}`)
			})
			req := request(tc.path, tc.args...)
			req.Flags, req.Body, req.Endpoint = tc.flags, json.RawMessage(tc.body), server.URL
			if tc.body == "" {
				req.Body = nil
			}
			if _, err := auth.New(server.Client()).Execute(t.Context(), approve(req)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUpdatePreservesOmittedOptions(t *testing.T) {
	const current = `{"data":{"id":"client-id","name":"Original","description":"Keep me","public":true,"trusted":true,"redirectUris":["https://example.com"],"postLogoutRedirectUris":["https://example.com/logout"],"scopes":["auth:read"],"metadata":{"id":"9007199254740993"},"secrets":[{"id":"secret","clear":"never-send"}],"serverOnly":9007199254740993}}`
	for _, tc := range []struct {
		name    string
		flags   map[string]string
		changed map[string]bool
		body    string
		changes string
	}{
		{name: "omitted host defaults", flags: map[string]string{"description": "Changed", "public": "false", "trusted": "false", "client-scopes": "", "redirect-uri": "", "post-logout-redirect-uri": "", "name": ""}, changed: map[string]bool{"description": true}, changes: `{"description":"Changed"}`},
		{name: "explicit false", flags: map[string]string{"public": "false", "trusted": "false"}, changed: map[string]bool{"public": true, "trusted": true}, changes: `{"public":false,"trusted":false}`},
		{name: "direct SDK false", flags: map[string]string{"public": "false"}, changes: `{"public":false}`},
		{name: "explicit empty", flags: map[string]string{"description": "", "client-scopes": "", "redirect-uri": "", "post-logout-redirect-uri": ""}, changed: map[string]bool{"description": true, "client-scopes": true, "redirect-uri": true, "post-logout-redirect-uri": true}, changes: `{"description":"","scopes":[],"redirectUris":[],"postLogoutRedirectUris":[]}`},
		{name: "direct empty", flags: map[string]string{"description": ""}, changes: `{"description":""}`},
		{name: "rename flag", flags: map[string]string{"name": "Renamed"}, changes: `{"name":"Renamed"}`},
		{name: "false string is a value", flags: map[string]string{"description": "false"}, changed: map[string]bool{}, changes: `{"description":"false"}`},
		{name: "CSV update", flags: map[string]string{"client-scopes": "auth:write,ledger:write"}, changes: `{"scopes":["auth:write","ledger:write"]}`},
		{name: "body patch", body: `{"public":false,"redirectUris":[],"metadata":{}}`, changes: `{"public":false,"redirectUris":[],"metadata":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := updateFixture(t, current, tc.changes)
			req := request("auth clients update", "client-id")
			req.Flags, req.ChangedFlags, req.Endpoint = tc.flags, tc.changed, server.URL+"/api/auth"
			if tc.body != "" {
				req.Body = json.RawMessage(tc.body)
			}
			req = approve(req)
			before := cloneRequest(req)
			response, err := auth.New(server.Client()).Execute(t.Context(), req)
			if err != nil || calls.Load() != 2 || string(response.Data) != `{"data":{"quantity":9007199254740993}}` {
				t.Fatalf("result=%s; err=%v; calls=%d", response.Data, err, calls.Load())
			}
			assertRequestUnchanged(t, req, before)
		})
	}
}

func TestCancellationDuringUpdateReadPreventsPut(t *testing.T) {
	entered := make(chan struct{})
	var reads, writes atomic.Int32
	server := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reads.Add(1)
		close(entered)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := approve(request("auth clients update", "client"))
	req.Flags["description"], req.Endpoint = "changed", server.URL
	finished := make(chan error, 1)
	go func() {
		_, err := auth.New(server.Client()).Execute(ctx, req)
		finished <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("update never started its read")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("update did not stop after cancellation")
	}
	if reads.Load() != 1 || writes.Load() != 0 {
		t.Fatalf("reads = %d; writes = %d", reads.Load(), writes.Load())
	}
}

func TestDirectSDKValidationBeforeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, path, body, message string
		args                      []string
		flags                     map[string]string
		changed                   map[string]bool
	}{
		{name: "unknown command", path: "auth clients rotate", message: "unknown plugin command"},
		{name: "non executable root", path: "auth", message: "not executable"},
		{name: "missing ID", path: "auth clients show", message: "arguments"},
		{name: "extra argument", path: "auth users list", args: []string{"extra"}, message: "arguments"},
		{name: "unknown flag", path: "auth clients list", flags: map[string]string{"page-size": "100"}, message: "unknown plugin flag"},
		{name: "unknown changed flag", path: "auth clients list", changed: map[string]bool{"bogus": true}, message: "unknown plugin flag"},
		{name: "empty ID", path: "auth users show", args: []string{" "}, message: "identifier"},
		{name: "dot ID", path: "auth clients show", args: []string{"."}, message: "identifier"},
		{name: "double dot secret", path: "auth clients secrets delete", args: []string{"client", ".."}, flags: map[string]string{"confirm": "true"}, message: "identifier"},
		{name: "unconfirmed client delete", path: "auth clients delete", args: []string{"client"}, message: "requires --confirm"},
		{name: "refused secret delete", path: "auth clients secrets delete", args: []string{"client", "secret"}, flags: map[string]string{"confirm": "false"}, message: "requires --confirm"},
		{name: "invalid confirm", path: "auth clients delete", args: []string{"client"}, flags: map[string]string{"confirm": "yes"}, message: "invalid --confirm"},
		{name: "invalid bool", path: "auth clients create", args: []string{"app"}, flags: map[string]string{"confirm": "true", "public": "yes"}, message: "invalid --public"},
		{name: "missing name", path: "auth clients create", flags: map[string]string{"confirm": "true"}, message: "name is required"},
		{name: "blank name", path: "auth clients create", args: []string{" "}, flags: map[string]string{"confirm": "true"}, message: "name must not be empty"},
		{name: "duplicate name", path: "auth clients create", args: []string{"app"}, flags: map[string]string{"confirm": "true", "name": "other"}, message: "only one"},
		{name: "duplicate body field", path: "auth clients create", body: `{"name":"app","public":true}`, flags: map[string]string{"confirm": "true", "public": "false"}, message: "either --data or --public"},
		{name: "invalid CSV", path: "auth clients create", args: []string{"app"}, flags: map[string]string{"confirm": "true", "client-scopes": `"unterminated`}, message: "valid CSV"},
		{name: "multiple CSV records", path: "auth clients create", args: []string{"app"}, flags: map[string]string{"confirm": "true", "client-scopes": "one\ntwo"}, message: "valid CSV"},
		{name: "core OAuth scopes rejected", path: "auth clients create", args: []string{"app"}, flags: map[string]string{"confirm": "true", "scopes": "auth:read"}, message: "unknown plugin flag"},
		{name: "host body required", path: "auth clients create", flags: map[string]string{"confirm": "true", "data": "@file"}, message: "decoded into Body"},
		{name: "invalid JSON", path: "auth clients create", body: `{"name":`, flags: map[string]string{"confirm": "true"}, message: "valid JSON"},
		{name: "oversized JSON", path: "auth clients create", body: `{"name":"` + strings.Repeat("a", 4<<20) + `"}`, flags: map[string]string{"confirm": "true"}, message: "4 MiB"},
		{name: "unexpected read body", path: "auth clients list", body: `{}`, message: "does not accept"},
		{name: "unexpected delete body", path: "auth clients delete", args: []string{"client"}, body: `{}`, flags: map[string]string{"confirm": "true"}, message: "does not accept"},
		{name: "null body", path: "auth clients create", body: `null`, flags: map[string]string{"confirm": "true"}, message: "JSON object"},
		{name: "array body", path: "auth clients create", body: `[]`, flags: map[string]string{"confirm": "true"}, message: "JSON object"},
		{name: "empty update", path: "auth clients update", args: []string{"client"}, flags: map[string]string{"confirm": "true"}, message: "at least one"},
		{name: "missing secret name", path: "auth clients secrets create", args: []string{"client"}, flags: map[string]string{"confirm": "true"}, message: "name is required"},
		{name: "secret extra field", path: "auth clients secrets create", args: []string{"client"}, body: `{"name":"secret","public":true}`, flags: map[string]string{"confirm": "true"}, message: "unsupported secret option"},
		{name: "unknown payload field", path: "auth clients create", body: `{"name":"app","id":"cannot-set"}`, flags: map[string]string{"confirm": "true"}, message: "unsupported client option"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := fixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			req := request(tc.path, tc.args...)
			req.Flags, req.ChangedFlags, req.Endpoint = tc.flags, tc.changed, server.URL
			if tc.body != "" {
				req.Body = json.RawMessage(tc.body)
			}
			_, err := auth.New(server.Client()).Execute(t.Context(), req)
			if err == nil || !strings.Contains(err.Error(), tc.message) || calls.Load() != 0 {
				t.Fatalf("error = %v; expected %q; calls = %d", err, tc.message, calls.Load())
			}
		})
	}
}

func TestPayloadTypes(t *testing.T) {
	for _, field := range []string{
		`"name":null`, `"name":9`, `"name":" "`, `"description":null`, `"description":false`,
		`"public":"false"`, `"trusted":null`, `"redirectUris":{}`, `"redirectUris":[null]`,
		`"scopes":[9]`, `"postLogoutRedirectUris":null`, `"metadata":[]`, `"metadata":{"n":9007199254740993}`, `"metadata":{"n":null}`,
	} {
		t.Run(field, func(t *testing.T) {
			var calls atomic.Int32
			server := fixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			req := approve(request("auth clients update", "client"))
			req.Body, req.Endpoint = json.RawMessage("{"+field+"}"), server.URL
			if _, err := auth.New(server.Client()).Execute(t.Context(), req); err == nil || calls.Load() != 0 {
				t.Fatalf("invalid payload was accepted: error = %v; calls = %d", err, calls.Load())
			}
		})
	}
}

func TestManifestOfflineAndSDKContract(t *testing.T) {
	manifest, err := auth.New(nil).GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip pluginsdk.Manifest
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Name != "auth" || roundTrip.Version != "1.0.0" || roundTrip.Service != "auth" || roundTrip.Root.Target != "stack" || roundTrip.Root.Service != "auth" {
		t.Fatalf("manifest = %s", encoded)
	}
	wantPaths := []string{"auth clients create", "auth clients list", "auth clients show", "auth clients update", "auth clients delete", "auth clients secrets create", "auth clients secrets delete", "auth clients users list", "auth clients users show", "auth users list", "auth users show"}
	gotPaths := manifestCommandPaths(t, roundTrip, roundTrip.Root, nil)
	slices.Sort(gotPaths)
	slices.Sort(wantPaths)
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("paths = %v, want %v", gotPaths, wantPaths)
	}
}

func TestFailures(t *testing.T) {
	for _, tc := range []failureCase{
		{name: "empty delete", path: "auth clients delete", args: []string{"client"}, status: 204},
		{name: "structured failure", path: "auth clients delete", args: []string{"client"}, status: 409, response: `{"errorCode":"IN_USE","errorMessage":"client is in use","quantity":9007199254740993}`, message: "IN_USE"},
		{name: "malformed response", path: "auth users list", status: 200, response: "{", message: "invalid JSON"},
		{name: "read failure blocks update", path: "auth clients update", args: []string{"client"}, status: 404, response: `{"errorCode":"NOT_FOUND","errorMessage":"missing"}`, message: "NOT_FOUND"},
		{name: "invalid read envelope", path: "auth clients update", args: []string{"client"}, status: 200, response: `{"data":[]}`, message: "data object"},
		{name: "missing read envelope", path: "auth clients update", args: []string{"client"}, status: 200, response: `{}`, message: "data object"},
		{name: "invalid read options", path: "auth clients update", args: []string{"client"}, status: 200, response: `{"data":{"name":42}}`, message: "invalid client options"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runFailureCase(t, tc)
		})
	}
}

func TestCancellationBeforeExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := auth.New(nil).GetManifest(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("manifest cancellation = %v", err)
	}
	if _, err := auth.New(nil).Execute(ctx, request("auth users list")); !errors.Is(err, context.Canceled) {
		t.Fatalf("execution cancellation = %v", err)
	}
}

func TestInvalidClientAndEndpoint(t *testing.T) {
	if _, err := auth.New(nil).Execute(t.Context(), request("auth users list")); err == nil || !strings.Contains(err.Error(), "HTTP client") {
		t.Fatalf("nil client = %v", err)
	}
	req := request("auth users list")
	req.Endpoint = "https://user:password@example.com/auth"
	if _, err := auth.New(http.DefaultClient).Execute(t.Context(), req); err == nil || !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("invalid endpoint = %v", err)
	}
}
