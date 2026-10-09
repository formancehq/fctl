package connectivity

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

//nolint:gocognit // Each route checks method, path, body, media type, defaults and response against the pinned OpenAPI contract.
func TestOpenAPIContract(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/operations.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Operations []struct{ ID, Method, Path string }
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Operations) != len(operations) {
		t.Fatal("OpenAPI operation coverage changed")
	}
	for _, expected := range contract.Operations {
		t.Run(expected.ID, func(t *testing.T) {
			i := slices.IndexFunc(operations, func(op operation) bool { return op.id == expected.ID })
			if i < 0 {
				t.Fatal("missing OpenAPI operation")
			}
			op := operations[i]
			req := pluginsdk.ExecuteRequest{CommandPath: strings.Fields("connectivity " + op.command), Endpoint: "https://api.example/prefix"}
			if expected.Method == "DELETE" || expected.Method == "PUT" {
				req.Flags = map[string]string{"confirm": "true"}
			}
			if strings.Contains(expected.Path, "{name}") {
				req.Args = []string{"stripe"}
			}
			if strings.Contains(expected.Path, "{version}") {
				req.Args = append(req.Args, "v1.2.3+build")
			}
			switch expected.Method {
			case "POST":
				req.Body = json.RawMessage(`{"name":"stripe","spec":{"connector":"stripe","ledger":"books","startSequence":9007199254740993}}`)
			case "PUT":
				req.Body = json.RawMessage(`{"spec":{"connector":"stripe","ledger":"books"}}`)
			case "PATCH":
				req.Body = json.RawMessage(`{"suspend":false,"version":null,"config":{"env":{"COUNT":{"value":"9007199254740993"}}}}`)
			}
			calls := 0
			p := New(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				path := strings.NewReplacer("{name}", "stripe", "{version}", "v1.2.3+build").Replace(expected.Path)
				if r.Method != expected.Method || r.URL.Path != "/prefix"+path {
					t.Errorf("request %s %s, want %s %s", r.Method, r.URL.Path, expected.Method, path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != string(req.Body) {
					t.Errorf("body=%s err=%v", body, err)
				}
				if req.Body != nil {
					contentType := "application/json"
					if expected.Method == "PATCH" {
						contentType = "application/merge-patch+json"
					}
					if r.Header.Get("Content-Type") != contentType {
						t.Error("incorrect write content type")
					}
				}
				if op.list && r.URL.Query().Get("pageSize") != "15" {
					t.Error("page-size default missing")
				}
				status, response := http.StatusOK, `{"data":{"sequence":9007199254740993}}`
				if expected.Method == "POST" {
					status = http.StatusCreated
				}
				if expected.Method == "DELETE" {
					status, response = http.StatusNoContent, ""
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
			})})
			result, err := p.Execute(t.Context(), req)
			want := `{"data":{"sequence":9007199254740993}}`
			if expected.Method == "DELETE" {
				want = "null"
			}
			if err != nil || string(result.Data) != want || calls != 1 {
				t.Fatalf("result=%s calls=%d err=%v", result.Data, calls, err)
			}
		})
	}
}

func TestRejectsInvalidInputBeforeHTTP(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, command, body string
		args                []string
		flags               map[string]string
	}{
		{"name", "instances show", "", []string{"../other"}, nil},
		{"version", "connectors versions show", "", []string{"stripe", ".."}, nil},
		{"page zero", "instances list", "", nil, map[string]string{"page-size": "0"}},
		{"page too large", "connectors list", "", nil, map[string]string{"page-size": "101"}},
		{"bad query", "connectors facets", "[]", nil, map[string]string{"query": "[]"}},
		{"missing data", "instances create", "", nil, nil},
		{"null data", "instances patch", "null", []string{"stripe"}, nil},
		{"missing spec", "instances create", `{"name":"stripe"}`, nil, nil},
		{"missing connector", "instances create", `{"name":"stripe","spec":{"ledger":"books"}}`, nil, nil},
		{"missing ledger", "instances create", `{"name":"stripe","spec":{"connector":"stripe"}}`, nil, nil},
		{"invalid labels", "instances create", `{"name":"stripe","labels":{"n":3},"spec":{"connector":"stripe","ledger":"books"}}`, nil, nil},
		{"negative sequence", "instances patch", `{"startSequence":-1}`, []string{"stripe"}, nil},
		{"fraction", "instances patch", `{"startSequence":1.5}`, []string{"stripe"}, nil},
		{"overflow", "instances patch", `{"startSequence":9223372036854775808}`, []string{"stripe"}, nil},
		{"wrong suspend", "instances patch", `{"suspend":"false"}`, []string{"stripe"}, nil},
		{"oversized", "instances patch", `{"note":"` + strings.Repeat("x", 1<<20) + `"}`, []string{"stripe"}, nil},
		{"missing confirm", "instances delete", "", []string{"stripe"}, nil},
		{"replace confirm", "instances replace", `{"spec":{}}`, []string{"stripe"}, nil},
		{"unexpected flag", "info", "", nil, map[string]string{"oops": "true"}},
		{"body on GET", "info", `{}`, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid input reached HTTP"); return nil, nil })})
			_, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: strings.Fields("connectivity " + tc.command), Args: tc.args, Flags: tc.flags, Body: json.RawMessage(tc.body), Endpoint: "https://api.example"})
			if err == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}

func TestCursorQueryAndVersionEscaping(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"connectivity", "connectors", "list"}, {"connectivity", "connectors", "versions", "show"}} {
		request := pluginsdk.ExecuteRequest{CommandPath: args, Endpoint: "https://api.example/prefix"}
		if args[len(args)-1] == "list" {
			request.Flags = map[string]string{"page-size": "100", "cursor": "a+/=?%&", "query": `{"$match":{"spec.category":"bank"}}`}
		} else {
			request.Args = []string{"stripe", "v1/x%2Fy"}
		}
		p := New(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			if request.Flags != nil {
				if r.URL.Query().Get("cursor") != request.Flags["cursor"] || r.URL.Query().Get("query") != request.Flags["query"] {
					t.Error("query or opaque cursor changed")
				}
			} else if r.URL.EscapedPath() != "/prefix/connectors/stripe/versions/v1%2Fx%252Fy" {
				t.Error("version must be encoded exactly once")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		})})
		if _, err := p.Execute(t.Context(), request); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStructuredServiceErrors(t *testing.T) {
	t.Parallel()
	for _, status := range []int{400, 401, 403, 404, 409, 422, 500, 503} {
		calls := 0
		p := New(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":"connector_immutable","message":"create a new instance","details":{"current":"stripe"}}`))}, nil
		})})
		response, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"connectivity", "instances", "patch"}, Args: []string{"stripe"}, Body: json.RawMessage(`{"suspend":true}`), Endpoint: "https://api.example"})
		var failure *httpclient.Error
		if !errors.As(err, &failure) || failure.StatusCode != status || failure.Code != "connector_immutable" || !strings.Contains(err.Error(), "create a new instance") || response.Data != nil || calls != 1 {
			t.Fatalf("error contract: data=%s err=%v calls=%d", response.Data, err, calls)
		}
	}
}
