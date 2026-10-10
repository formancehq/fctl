package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
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

func readRequestBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read request body: %v", err)
	}
	return body
}

func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func assertRequestRoute(t *testing.T, r *http.Request, method, route string) {
	t.Helper()
	if r.Method != method || r.URL.EscapedPath() != route || r.URL.RawQuery != "" {
		t.Errorf("request = %s %s, want %s %s without query", r.Method, r.URL, method, route)
	}
}

func assertRequestBody(t *testing.T, r *http.Request, want string) {
	t.Helper()
	body := readRequestBody(t, r)
	if want == "" {
		if len(body) != 0 {
			t.Errorf("unexpected request body: %s", body)
		}
		return
	}
	assertJSON(t, body, want)
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
	}
}

func historicalRouteFixture(t *testing.T, method, route, body, response string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assertRequestRoute(t, r, method, "/gateway/api/auth"+route)
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		assertRequestBody(t, r, body)
		writeResponse(t, w, response)
	})
	return server, &calls
}

func updateFixture(t *testing.T, current, changes string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			assertRequestRoute(t, r, http.MethodGet, "/api/auth/clients/client-id")
			writeResponse(t, w, current)
			return
		}
		assertRequestRoute(t, r, http.MethodPut, "/api/auth/clients/client-id")
		if call != 2 {
			t.Errorf("unexpected call %d", call)
		}
		assertUpdatedFields(t, readRequestBody(t, r), current, changes)
		writeResponse(t, w, `{"data":{"quantity":9007199254740993}}`)
	})
	return server, &calls
}

func assertUpdatedFields(t *testing.T, body []byte, current, changes string) {
	t.Helper()
	fields := object(t, body)
	want := object(t, object(t, []byte(current))["data"])
	delete(want, "id")
	delete(want, "secrets")
	delete(want, "serverOnly")
	maps.Copy(want, object(t, []byte(changes)))
	if len(fields) != len(want) {
		t.Errorf("PUT fields = %s", body)
	}
	for key, value := range want {
		assertJSON(t, fields[key], string(value))
	}
}

func cloneRequest(req pluginsdk.ExecuteRequest) pluginsdk.ExecuteRequest {
	req.Flags = maps.Clone(req.Flags)
	req.ChangedFlags = maps.Clone(req.ChangedFlags)
	req.Body = bytes.Clone(req.Body)
	return req
}

func assertRequestUnchanged(t *testing.T, actual, before pluginsdk.ExecuteRequest) {
	t.Helper()
	if !reflect.DeepEqual(actual.Flags, before.Flags) || !reflect.DeepEqual(actual.ChangedFlags, before.ChangedFlags) || !bytes.Equal(actual.Body, before.Body) {
		t.Fatal("execution mutated the caller's request")
	}
}

func manifestCommandPaths(t *testing.T, manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, parent []string) []string {
	t.Helper()
	path := append(slices.Clone(parent), pluginsdk.CommandName(command))
	var paths []string
	if command.Runnable {
		paths = append(paths, strings.Join(path, " "))
		assertCommandContract(t, manifest, command, path)
	}
	for _, child := range command.Subcommands {
		paths = append(paths, manifestCommandPaths(t, manifest, child, path)...)
	}
	return paths
}

func assertCommandContract(t *testing.T, manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, path []string) {
	t.Helper()
	service, err := pluginsdk.CommandService(manifest, path)
	if err != nil || service != "auth" {
		t.Fatalf("service for %v = %q: %v", path, service, err)
	}
	for _, input := range command.Inputs {
		assertInputContract(t, manifest, command, input, path)
	}
	if command.Confirm {
		assertConfirmationRequired(t, manifest, command, path)
	}
}

func assertInputContract(t *testing.T, manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, input pluginsdk.InputSpec, path []string) {
	t.Helper()
	if input.Argument != nil && (*input.Argument < 0 || *input.Argument >= command.Args.Max) {
		t.Fatalf("invalid argument input for %v", path)
	}
	if input.Source == nil {
		return
	}
	read, err := pluginsdk.FindCommand(manifest, input.Source.CommandPath)
	if err != nil || !read.Runnable || pluginsdk.CommandName(read) != "list" {
		t.Fatalf("invalid selector source for %v: %v", path, err)
	}
}

func assertConfirmationRequired(t *testing.T, manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, path []string) {
	t.Helper()
	req := request(strings.Join(path, " "), make([]string, command.Args.Min)...)
	if _, err := pluginsdk.NormalizeRequest(manifest, req); err == nil {
		t.Fatalf("unconfirmed mutation accepted: %v", path)
	}
	if _, err := pluginsdk.NormalizeRequest(manifest, approve(req)); err != nil {
		t.Fatal(err)
	}
}

type failureCase struct {
	name, path, response, message string
	status                        int
	args                          []string
}

func runFailureCase(t *testing.T, tc failureCase) {
	t.Helper()
	var calls atomic.Int32
	server := fixture(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(tc.status)
		writeResponse(t, w, tc.response)
	})
	req := request(tc.path, tc.args...)
	if !strings.HasSuffix(tc.path, "list") {
		req = approve(req)
	}
	if strings.HasSuffix(tc.path, "update") {
		req.Flags["description"] = "update"
	}
	req.Endpoint = server.URL
	response, err := auth.New(server.Client()).Execute(t.Context(), req)
	assertFailureResponse(t, response, err, tc)
	if calls.Load() != 1 {
		t.Fatalf("calls = %d; failed operations must not retry", calls.Load())
	}
}

func assertFailureResponse(t *testing.T, response pluginsdk.ExecuteResponse, err error, tc failureCase) {
	t.Helper()
	if tc.message == "" {
		if err != nil || string(response.Data) != "null" {
			t.Fatalf("response = %s: %v", response.Data, err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), tc.message) {
		t.Fatalf("error = %v, want %s", err, tc.message)
	}
	if tc.name == "structured failure" {
		assertHTTPFailure(t, err, tc.response, tc.status)
	}
}

func assertHTTPFailure(t *testing.T, err error, body string, status int) {
	t.Helper()
	failure, ok := errors.AsType[*httpclient.Error](err)
	if !ok || string(failure.Body) != body || failure.StatusCode != status {
		t.Fatalf("structured error = %v", err)
	}
}
