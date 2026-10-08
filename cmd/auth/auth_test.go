package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func execute(t *testing.T, args []string, input string, transport roundTrip) (string, error) {
	t.Helper()
	conn, err := api.New("https://example.test/api/auth", &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer injected")
		return transport(req)
	})})
	if err != nil {
		t.Fatal(err)
	}
	cmd := NewCommand(command.Runtime{Client: func(ctx context.Context, service string) (*api.Client, error) {
		if service != "auth" {
			t.Fatalf("service = %s", service)
		}
		if ctx != t.Context() {
			t.Fatal("lost command context")
		}
		return conn, nil
	}})
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	err = cmd.ExecuteContext(t.Context())
	return output.String(), err
}
func reply(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestCommands(t *testing.T) {
	tests := []commandTest{
		{"info", []string{"info"}, "GET", "/_info", "", `{"version":"2.5.0","precision":9007199254740993}`, `9007199254740993`},
		{"discovery", []string{"discovery"}, "GET", "/.well-known/openid-configuration", "", `{"issuer":"https://example.test","token_endpoint":"https://example.test/oauth/token"}`, `token_endpoint`},
		{"clients list", []string{"clients", "list"}, "GET", "/clients", "", `{"data":[]}`, `"data"`},
		{"clients show", []string{"clients", "show", "c1"}, "GET", "/clients/c1", "", `{"data":{"id":"c1","name":"test","extra":9007199254740993}}`, `9007199254740993`},
		{"clients create", []string{"clients", "create", "--data", `{"name":"new","public":false,"scopes":["auth:read"],"metadata":{"env":"test"}}`}, "POST", "/clients", `{"name":"new","public":false,"scopes":["auth:read"],"metadata":{"env":"test"}}`, `{"data":{"id":"c1","name":"new"}}`, `"new"`},
		{"clients delete", []string{"clients", "delete", "c1", "--confirm"}, "DELETE", "/clients/c1", "", "", "null"},
		{"secrets create", []string{"clients", "secrets", "create", "c1", "--data", "-"}, "POST", "/clients/c1/secrets", `{"name":"secret","metadata":{"env":"test"}}`, `{"data":{"id":"s1","name":"secret","clear":"one-time"}}`, `one-time`},
		{"secrets delete", []string{"clients", "secrets", "delete", "c1", "s1", "--confirm"}, "DELETE", "/clients/c1/secrets/s1", "", "", "null"},
		{"secrets list", []string{"clients", "secrets", "list", "c1"}, "GET", "/clients/c1", "", `{"data":{"id":"c1","name":"test","secrets":[{"id":"s1","name":"secret","lastDigits":"1234","extra":9007199254740993}]}}`, `9007199254740993`},
		{"users list", []string{"users", "list"}, "GET", "/users", "", `{"data":[]}`, `"data"`},
		{"users show", []string{"users", "show", "u1"}, "GET", "/users/u1", "", `{"data":{"id":"u1","subject":"sub","email":"user@example.test"}}`, `user@example.test`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCommandTest(t, tt)
		})
	}
}
func assertJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var a, b any
	da := json.NewDecoder(bytes.NewReader(got))
	da.UseNumber()
	db := json.NewDecoder(bytes.NewReader(want))
	db.UseNumber()
	if err := da.Decode(&a); err != nil {
		t.Fatal(err)
	}
	if err := db.Decode(&b); err != nil {
		t.Fatal(err)
	}
	aa, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(aa, bb) {
		t.Fatalf("JSON = %s; want %s", aa, bb)
	}
}

func TestUpdatePreservesAndClearsOptions(t *testing.T) {
	for _, patch := range []string{`{"description":"changed"}`, `{"public":false,"trusted":false,"redirectUris":[],"postLogoutRedirectUris":[],"scopes":[],"metadata":{}}`} {
		t.Run(patch, func(t *testing.T) {
			calls := 0
			current := `{"data":{"id":"c1","name":"original","description":"old","public":true,"trusted":true,"redirectUris":["https://example.test/cb"],"postLogoutRedirectUris":["https://example.test/bye"],"scopes":["auth:read"],"metadata":{"env":"test"},"secrets":[]}}`
			output, err := execute(t, []string{"clients", "update", "c1", "--confirm", "--data", patch}, "", func(req *http.Request) (*http.Response, error) {
				calls++
				return updateReply(t, req, calls, current, patch), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if calls != 2 || !strings.Contains(output, "original") {
				t.Fatalf("calls=%d output=%s", calls, output)
			}
		})
	}
}
func TestValidationBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{
		{"clients", "delete", "c1"}, {"clients", "update", "c1", "--data", `{"name":"x"}`}, {"clients", "secrets", "delete", "c1", "s1"},
		{"clients", "create", "--data", `{"unknown":true,"name":"x"}`}, {"clients", "create", "--data", `null`}, {"clients", "create", "--data", `{"name":""}`},
		{"clients", "create", "--data", `{"name":null}`}, {"clients", "update", "c1", "--confirm", "--data", `{"scopes":1}`},
		{"clients", "secrets", "create", "c1", "--data", `{}`}, {"clients", "create"},
		{"clients", "show", ".."},
		{"clients", "show", ""},
		{"users", "show", "."},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output, err := execute(t, args, "", func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network request"); return nil, nil })
			if err == nil {
				t.Fatal("expected validation error")
			}
			if output != "" {
				t.Fatalf("output = %s", output)
			}
		})
	}
}
func TestRequestErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{"unauthorized", 401, `{"errorCode":"UNAUTHORIZED","errorMessage":"denied"}`, nil},
		{"not found", 404, `{"errorCode":"NOT_FOUND","errorMessage":"missing"}`, nil},
		{"transport", 0, "", errors.New("offline")},
		{"invalid json", 200, `broken`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := execute(t, []string{"info"}, "", func(*http.Request) (*http.Response, error) {
				if test.err != nil {
					return nil, test.err
				}
				return reply(test.status, test.body), nil
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if output != "" {
				t.Fatalf("output = %s", output)
			}
		})
	}
}

func TestDataFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(`{"name":"from-file"}`), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := execute(t, []string{"clients", "create", "--data", "@" + path}, "", func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		assertJSON(t, body, []byte(`{"name":"from-file"}`))
		return reply(201, `{"data":{"id":"c1","name":"from-file"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "from-file") {
		t.Fatal(output)
	}
}
func TestEncodedIdentifier(t *testing.T) {
	_, err := execute(t, []string{"clients", "show", "a/b ?#%"}, "", func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/auth/clients/a%2Fb%20%3F%23%25" || req.URL.RawQuery != "" {
			t.Fatalf("URL = %s", req.URL)
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
		_, err := execute(t, []string{"clients", "update", "c1", "--confirm", "--data", `{"description":"changed"}`}, "", func(req *http.Request) (*http.Response, error) {
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
	output, err := execute(t, []string{"clients", "secrets", "list", "c1"}, "", func(*http.Request) (*http.Response, error) {
		return reply(200, `{"data":{"id":"c1","name":"test"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, []byte(output), []byte(`[]`))
}
func TestRuntimeFailure(t *testing.T) {
	offline := errors.New("connection unavailable")
	cmd := NewCommand(command.Runtime{Client: func(context.Context, string) (*api.Client, error) { return nil, offline }})
	cmd.SetArgs([]string{"info"})
	cmd.SetErr(io.Discard)
	if err := cmd.ExecuteContext(t.Context()); !errors.Is(err, offline) {
		t.Fatalf("err=%v", err)
	}
}

func checkUpdatedOptions(t *testing.T, body []byte, current, patch string) {
	t.Helper()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
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
	if err := json.Unmarshal([]byte(current), &envelope); err != nil {
		t.Fatal(err)
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal([]byte(patch), &changes); err != nil {
		t.Fatal(err)
	}
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

func checkOption(t *testing.T, key string, got, want json.RawMessage) {
	t.Helper()
	if len(got) == 0 {
		// The generated SDK omits empty collections; Auth decodes them as empty.
		if string(want) != "[]" && string(want) != "{}" {
			t.Fatalf("lost %s", key)
		}
		return
	}
	assertJSON(t, got, want)
}

type commandTest struct {
	name                               string
	args                               []string
	method, path, body, response, want string
}

func runCommandTest(t *testing.T, tt commandTest) {
	t.Helper()

	calls := 0
	output, err := execute(t, tt.args, tt.body, func(req *http.Request) (*http.Response, error) {
		calls++
		return commandReply(t, tt, req), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
	if !json.Valid([]byte(output)) || !strings.Contains(output, tt.want) {
		t.Fatalf("output = %s", output)
	}
	if tt.name == "secrets list" && strings.Contains(output, `"data"`) {
		t.Fatal("expected secret array")
	}
}

func commandReply(t *testing.T, tt commandTest, req *http.Request) *http.Response {
	t.Helper()
	if req.Method != tt.method || req.URL.Path != "/api/auth"+tt.path {
		t.Fatalf("request = %s %s", req.Method, req.URL)
	}
	if req.Header.Get("Authorization") != "Bearer injected" {
		t.Fatal("lost injected authentication")
	}
	if tt.body != "" {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		assertJSON(t, body, []byte(tt.body))
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
	}
	code := 200
	if tt.method == "POST" && tt.name != "secrets create" {
		code = 201
	}
	if tt.method == "DELETE" {
		code = 204
	}
	return reply(code, tt.response)
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
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	checkUpdatedOptions(t, body, current, patch)
	return reply(200, current)
}
