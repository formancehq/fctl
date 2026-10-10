// Package testutil contains strict HTTP fixtures shared by legacy service contract tests.
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type Exchange struct {
	Method, Path, Query, Body, Key, Response string
	Status                                   int
}

func Fixture(t *testing.T, exchanges []Exchange) (*httptest.Server, func()) {
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
		status := expected.Status
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)
		response := expected.Response
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

func AssertJSON(t *testing.T, actual, expected []byte) {
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

type Case struct {
	Name, Command string
	Args          []string
	Flags         map[string]string
	Changed       map[string]bool
	Body          string
	Exchanges     []Exchange
}

func RunCase(t *testing.T, factory func(*http.Client) pluginsdk.Plugin, tc Case) {
	t.Helper()
	server, verify := Fixture(t, tc.Exchanges)
	defer verify()
	path := strings.Fields(tc.Command)
	p := factory(server.Client())
	var body json.RawMessage
	if tc.Body != "" {
		body = json.RawMessage(tc.Body)
	}
	response, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: path, Args: tc.Args, Flags: tc.Flags, ChangedFlags: tc.Changed, Body: body, Endpoint: server.URL + "/gateway/service"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(response.Data) {
		t.Errorf("invalid response JSON: %s", response.Data)
	}
}

func RunFailure(t *testing.T, factory func(*http.Client) pluginsdk.Plugin, tc Case, want string) {
	t.Helper()
	server, verify := Fixture(t, tc.Exchanges)
	defer verify()
	var body json.RawMessage
	if tc.Body != "" {
		body = json.RawMessage(tc.Body)
	}
	_, err := factory(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{
		CommandPath: strings.Fields(tc.Command), Args: tc.Args, Flags: tc.Flags,
		ChangedFlags: tc.Changed, Body: body, Endpoint: server.URL + "/gateway/service",
	})
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected failure containing %q, got %v", want, err)
	}
}

func verifyRequest(t *testing.T, r *http.Request, expected Exchange) {
	t.Helper()
	if r.Method != expected.Method || r.URL.EscapedPath() != "/gateway/service"+expected.Path {
		t.Errorf("request: %s %s; expected %s /gateway/service%s", r.Method, r.URL.EscapedPath(), expected.Method, expected.Path)
	}
	query, err := url.ParseQuery(expected.Query)
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
	if expected.Body == "" {
		if len(body) > 0 {
			t.Errorf("unexpected body: %s", body)
		}
	} else {
		AssertJSON(t, body, []byte(expected.Body))
	}
	verifyHeaders(t, r, expected.Key, len(body) > 0)
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
