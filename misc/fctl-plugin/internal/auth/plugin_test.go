package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
)

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
