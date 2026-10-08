package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/formancehq/auth/pkg/client/models/components"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func execute(t *testing.T, req pluginsdk.ExecuteRequest, transport roundTrip) (pluginsdk.ExecuteResponse, error) {
	t.Helper()
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Context() != t.Context() {
			t.Fatal("lost request context")
		}
		r.Header.Set("Authorization", "Bearer injected")
		return transport(r)
	})}
	req.Endpoint = "https://example.test/api/auth/"
	return New(client).Execute(t.Context(), req)
}
func request(path string, args []string, body string, confirm bool) pluginsdk.ExecuteRequest {
	req := pluginsdk.ExecuteRequest{CommandPath: strings.Fields("auth " + path), Args: args, Flags: map[string]string{}, ChangedFlags: map[string]bool{}}
	if body != "" {
		req.Body = json.RawMessage(body)
		req.Flags["data"] = "-"
		req.ChangedFlags["data"] = true
	}
	if confirm {
		req.Flags["confirm"] = "true"
		req.ChangedFlags["confirm"] = true
	}
	return req
}
func reply(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

type commandTest struct {
	path                        string
	args                        []string
	body                        string
	confirm                     bool
	method, url, response, want string
}

func TestCommands(t *testing.T) {
	cases := []commandTest{
		{"info", nil, "", false, "GET", "/_info", `{"version":"2.5.0","precision":9007199254740993}`, "9007199254740993"},
		{"discovery", nil, "", false, "GET", "/.well-known/openid-configuration", `{"issuer":"https://example.test","token_endpoint":"https://example.test/oauth/token"}`, "token_endpoint"},
		{"clients list", nil, "", false, "GET", "/clients", `{"data":[]}`, `"data"`},
		{"clients show", []string{"c1"}, "", false, "GET", "/clients/c1", `{"data":{"id":"c1","name":"test","extra":9007199254740993}}`, "9007199254740993"},
		{"clients create", nil, `{"name":"new","public":false,"scopes":["auth:read"],"metadata":{"env":"test"}}`, false, "POST", "/clients", `{"data":{"id":"c1","name":"new"}}`, `"new"`},
		{"clients delete", []string{"c1"}, "", true, "DELETE", "/clients/c1", "", "null"},
		{"clients secrets create", []string{"c1"}, `{"name":"secret","metadata":{"env":"test"}}`, false, "POST", "/clients/c1/secrets", `{"data":{"id":"s1","name":"secret","clear":"one-time"}}`, "one-time"},
		{"clients secrets delete", []string{"c1", "s1"}, "", true, "DELETE", "/clients/c1/secrets/s1", "", "null"},
		{"clients secrets list", []string{"c1"}, "", false, "GET", "/clients/c1", `{"data":{"id":"c1","name":"test","secrets":[{"id":"s1","name":"secret","lastDigits":"1234","extra":9007199254740993}]}}`, "9007199254740993"},
		{"users list", nil, "", false, "GET", "/users", `{"data":[]}`, `"data"`},
		{"users show", []string{"u1"}, "", false, "GET", "/users/u1", `{"data":{"id":"u1","subject":"sub","email":"user@example.test"}}`, "user@example.test"},
	}
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) { testCommand(t, test) })
	}
}
func testCommand(t *testing.T, test commandTest) {
	t.Helper()
	calls := 0
	res, err := execute(t, request(test.path, test.args, test.body, test.confirm), func(req *http.Request) (*http.Response, error) { calls++; return commandReply(t, test, req), nil })
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if !json.Valid(res.Data) || !strings.Contains(string(res.Data), test.want) {
		t.Fatalf("data=%s", res.Data)
	}
	if test.path == "clients secrets list" && strings.Contains(string(res.Data), `"data"`) {
		t.Fatal("expected a secret array")
	}
}
func commandReply(t *testing.T, test commandTest, req *http.Request) *http.Response {
	t.Helper()
	if req.Method != test.method || req.URL.Path != "/api/auth"+test.url {
		t.Fatalf("request=%s %s", req.Method, req.URL)
	}
	if req.Header.Get("Authorization") != "Bearer injected" {
		t.Fatal("lost injected authentication")
	}
	if test.body != "" {
		assertJSON(t, readRequest(t, req), []byte(test.body))
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing content type")
		}
	}
	code := 200
	if test.path == "clients create" {
		code = 201
	}
	if test.method == "DELETE" {
		code = 204
	}
	return reply(code, test.response)
}
func readRequest(t *testing.T, req *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func assertJSON(t *testing.T, got, want []byte) {
	t.Helper()
	a := canonicalJSON(t, got)
	b := canonicalJSON(t, want)
	if !bytes.Equal(a, b) {
		t.Fatalf("JSON=%s; want %s", a, b)
	}
}
func canonicalJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestUpdatePreservesAndClearsOptions(t *testing.T) {
	current := `{"data":{"id":"c1","name":"original","description":"old","public":true,"trusted":true,"redirectUris":["https://example.test/cb"],"postLogoutRedirectUris":["https://example.test/bye"],"scopes":["auth:read"],"metadata":{"env":"test"},"secrets":[]}}`
	for _, patch := range []string{`{"description":"changed"}`, `{"public":false,"trusted":false,"redirectUris":[],"postLogoutRedirectUris":[],"scopes":[],"metadata":{}}`} {
		t.Run(patch, func(t *testing.T) {
			calls := 0
			res, err := execute(t, request("clients update", []string{"c1"}, patch, true), func(req *http.Request) (*http.Response, error) {
				calls++
				return updateReply(t, req, calls, current, patch), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if calls != 2 || !strings.Contains(string(res.Data), "original") {
				t.Fatalf("calls=%d data=%s", calls, res.Data)
			}
		})
	}
}
func updateReply(t *testing.T, req *http.Request, calls int, current, patch string) *http.Response {
	t.Helper()
	if req.URL.Path != "/api/auth/clients/c1" {
		t.Fatal(req.URL)
	}
	if calls == 1 {
		if req.Method != "GET" {
			t.Fatal(req.Method)
		}
		return reply(200, current)
	}
	if req.Method != "PUT" {
		t.Fatal(req.Method)
	}
	checkUpdatedOptions(t, readRequest(t, req), current, patch)
	return reply(200, current)
}
func checkUpdatedOptions(t *testing.T, body []byte, current, patch string) {
	t.Helper()
	var got map[string]json.RawMessage
	decode(t, body, &got)
	if _, ok := got["id"]; ok {
		t.Fatal("sent read-only id")
	}
	if _, ok := got["secrets"]; ok {
		t.Fatal("sent secrets")
	}
	if string(got["name"]) != `"original"` {
		t.Fatal("lost name")
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	decode(t, []byte(current), &envelope)
	var changes map[string]json.RawMessage
	decode(t, []byte(patch), &changes)
	for key, value := range envelope.Data {
		if key == "id" || key == "secrets" {
			continue
		}
		if v, ok := changes[key]; ok {
			value = v
		}
		checkOption(t, key, got[key], value)
	}
}
func decode(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}
func checkOption(t *testing.T, key string, got, want json.RawMessage) {
	t.Helper()
	if len(got) == 0 { // Generated omitempty fields decode as empty options on the server.
		if string(want) != "[]" && string(want) != "{}" {
			t.Fatalf("lost %s", key)
		}
		return
	}
	assertJSON(t, got, want)
}

// The host may authenticate before Execute; this fixture checks service HTTP only.
func TestValidationBeforeServiceRequest(t *testing.T) {
	cases := []pluginsdk.ExecuteRequest{
		request("clients delete", []string{"c1"}, "", false), request("clients update", []string{"c1"}, `{"name":"x"}`, false), request("clients secrets delete", []string{"c1", "s1"}, "", false),
		request("clients create", nil, `{"unknown":true,"name":"x"}`, false), request("clients create", nil, `null`, false), request("clients create", nil, `{"name":""}`, false), request("clients create", nil, `{"name":null}`, false),
		request("clients update", []string{"c1"}, `{"scopes":1}`, true), request("clients secrets create", []string{"c1"}, `{}`, false), request("clients create", nil, "", false),
		request("clients show", []string{".."}, "", false), request("users show", []string{""}, "", false), request("users show", []string{"."}, "", false), request("missing", nil, "", false), request("clients", nil, "", false), request("users show", nil, "", false),
	}
	for _, req := range cases {
		t.Run(strings.Join(req.CommandPath, " ")+string(req.Body), func(t *testing.T) {
			res, err := execute(t, req, func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network request"); return nil, nil })
			if err == nil || len(res.Data) > 0 {
				t.Fatalf("data=%s err=%v", res.Data, err)
			}
		})
	}
}
func TestFlagValidation(t *testing.T) {
	for _, flags := range []map[string]string{{"confirm": "invalid"}, {"unknown": "true"}} {
		req := request("clients delete", []string{"c1"}, "", true)
		req.Flags = flags
		_, err := execute(t, req, func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network request"); return nil, nil })
		if err == nil {
			t.Fatal("expected invalid flag error")
		}
	}
}
func TestErrorsLeaveDataEmpty(t *testing.T) {
	for _, status := range []int{401, 404, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := `{"errorCode":"FAIL","errorMessage":"denied","partial":9007199254740993}`
			res, err := execute(t, request("info", nil, "", false), func(*http.Request) (*http.Response, error) { return reply(status, body), nil })
			if err == nil {
				t.Fatal("expected error")
			}
			if len(res.Data) != 0 {
				t.Fatalf("error data=%s", res.Data)
			}
		})
	}
	offline := errors.New("offline")
	res, err := execute(t, request("info", nil, "", false), func(*http.Request) (*http.Response, error) { return nil, offline })
	if !errors.Is(err, offline) || res.Data != nil {
		t.Fatalf("data=%s err=%v", res.Data, err)
	}
}
func TestInvalidResponse(t *testing.T) {
	for _, code := range []int{200, 400} {
		_, err := execute(t, request("info", nil, "", false), func(*http.Request) (*http.Response, error) { return reply(code, "broken"), nil })
		if err == nil {
			t.Fatal("expected JSON error")
		}
	}
}
func TestEncodedIdentifier(t *testing.T) {
	_, err := execute(t, request("clients show", []string{"a/b ?#%"}, "", false), func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/auth/clients/a%2Fb%20%3F%23%25" || req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		return reply(200, `{"data":{"id":"c1","name":"test"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpdateReadFailureDoesNotWrite(t *testing.T) {
	for _, body := range []string{`{"errorCode":"NOT_FOUND","errorMessage":"missing"}`, `{}`} {
		calls := 0
		_, err := execute(t, request("clients update", []string{"c1"}, `{"description":"changed"}`, true), func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal("write after failed read")
			}
			code := 404
			if body == `{}` {
				code = 200
			}
			return reply(code, body), nil
		})
		if err == nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}
func TestEmptySecrets(t *testing.T) {
	res, err := execute(t, request("clients secrets list", []string{"c1"}, "", false), func(*http.Request) (*http.Response, error) {
		return reply(200, `{"data":{"id":"c1","name":"test"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, res.Data, []byte(`[]`))
}
func TestManifest(t *testing.T) {
	m, err := New(nil).GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "auth" || m.Service != "auth" || m.Version == "" || m.ProtocolVersion != pluginsdk.ProtocolVersion {
		t.Fatalf("manifest=%+v", m)
	}
	update, err := pluginsdk.FindCommand(m, []string{"auth", "clients", "update"})
	if err != nil {
		t.Fatal(err)
	}
	if !update.Confirm || !strings.Contains(update.Long, "preserved") {
		t.Fatal("update contract missing")
	}
	create, err := pluginsdk.FindCommand(m, []string{"auth", "clients", "create"})
	if err != nil {
		t.Fatal(err)
	}
	if len(create.Flags) != 1 || !create.Flags[0].Required || !create.Flags[0].Body || create.Flags[0].Type != "string" {
		t.Fatal("body contract missing")
	}
	// Each call returns an independent manifest, safe for concurrent host adapters.
	m.Root.Subcommands[0].Use = "changed"
	fresh, err := New(nil).GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Root.Subcommands[0].Use != "info" {
		t.Fatal("shared manifest state")
	}
}
func TestConnectionValidation(t *testing.T) {
	_, err := New(nil).Execute(t.Context(), request("info", nil, "", false))
	if err == nil {
		t.Fatal("expected missing client error")
	}
	req := request("info", nil, "", false)
	req.Endpoint = "https://user:secret@example.test"
	_, err = New(&http.Client{}).Execute(t.Context(), req)
	if err == nil {
		t.Fatal("expected endpoint error")
	}
}

func TestManifestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	m, err := New(nil).GetManifest(ctx)
	if !errors.Is(err, context.Canceled) || m.Name != "" {
		t.Fatalf("manifest=%+v err=%v", m, err)
	}
}
func TestProvidedBodyWithoutDataFlag(t *testing.T) {
	req := request("clients create", nil, `{"name":"direct"}`, false)
	req.Flags = nil
	req.ChangedFlags = nil
	res, err := execute(t, req, func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/api/auth/clients" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		assertJSON(t, readRequest(t, r), []byte(`{"name":"direct"}`))
		return reply(201, `{"data":{"id":"c1","name":"direct"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.Data), "direct") {
		t.Fatal(string(res.Data))
	}
}

type responseFixture struct{ metadata components.HTTPMetadata }

func (r responseFixture) GetHTTPMeta() components.HTTPMetadata { return r.metadata }

type unreadBody struct{ touched bool }

func (b *unreadBody) Read([]byte) (int, error) {
	b.touched = true
	return 0, errors.New("response must not be read")
}
func (b *unreadBody) Close() error {
	b.touched = true
	return errors.New("response must not be processed")
}
func TestResultErrorDoesNotProcessResponse(t *testing.T) {
	body := &unreadBody{}
	failure := errors.New("operation failed")
	res := responseFixture{metadata: components.HTTPMetadata{Response: &http.Response{StatusCode: 400, Body: body}}}
	data, err := result(res, failure)
	if len(data) != 0 || !errors.Is(err, failure) || body.touched {
		t.Fatalf("data=%s err=%v response touched=%v", data, err, body.touched)
	}
}

func TestResultMissingResponse(t *testing.T) {
	var absent *responseFixture
	for _, res := range []response{nil, absent} {
		data, err := result(res, nil)
		if len(data) != 0 || err == nil {
			t.Fatalf("missing response: data=%s err=%v", data, err)
		}
	}
}
