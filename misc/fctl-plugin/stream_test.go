package legacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type streamRoundTripper func(*http.Request) (*http.Response, error)

func (f streamRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func streamHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func requireStreaming(t *testing.T, p pluginsdk.Plugin) pluginsdk.StreamingPlugin {
	t.Helper()
	streaming, ok := p.(pluginsdk.StreamingPlugin)
	if !ok {
		t.Fatal("wrapper must implement StreamingPlugin")
	}
	return streaming
}

type guardedInput struct {
	t       *testing.T
	checked *bool
	reads   int
	reader  io.Reader
}

func (g *guardedInput) Read(p []byte) (int, error) {
	g.reads++
	if !*g.checked {
		g.t.Fatal("input consumed before compatibility check")
	}
	return g.reader.Read(p)
}

type guardedOutput struct {
	t       *testing.T
	checked *bool
	writes  int
	bytes.Buffer
}

func (g *guardedOutput) Write(p []byte) (int, error) {
	g.writes++
	if !*g.checked {
		g.t.Fatal("output used before compatibility check")
	}
	return g.Buffer.Write(p)
}

//nolint:gocognit // Exercise both public facades and directions against the compatibility matrix.
func TestStreamWrappersCompatibilityBeforeIO(t *testing.T) {
	for _, bundle := range []bool{false, true} {
		for _, command := range []string{"import", "export"} {
			for _, target := range []struct {
				version, line string
				allowed       bool
			}{{"2.4.15", "v3.2", true}, {"3.0.0-beta.10", "v3.2", false}, {"2.4.15", "v4.0-beta", false}, {"dev", "", false}} {
				t.Run(fmt.Sprintf("bundle=%t/%s/%s/%s", bundle, command, target.version, target.line), func(t *testing.T) {
					checked := false
					calls := 0
					client := &http.Client{Transport: streamRoundTripper(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.URL.Path == "/_info" {
							checked = true
							return streamHTTPResponse(200, `{"version":"`+target.version+`"}`), nil
						}
						if !target.allowed {
							t.Fatal("compatibility guard allowed a service operation")
						}
						if command == "import" {
							data, err := io.ReadAll(r.Body)
							if err != nil || string(data) != "{\"id\":1}\n" {
								t.Fatalf("import %s: %v", data, err)
							}
							return streamHTTPResponse(204, ""), nil
						}
						return streamHTTPResponse(200, "{\"id\":1}"), nil
					})}
					p := NewService("ledger", client)
					path := []string{"l", command} // Preserve the historical service alias.
					if bundle {
						p = New(client)
						path = append([]string{"legacy"}, path...)
					}
					r := pluginsdk.ExecuteRequest{Endpoint: "https://ledger.example", CommandPath: path, Context: map[string]string{"stack-version": target.line}}
					input := &guardedInput{t: t, checked: &checked, reader: strings.NewReader("{\"id\":1}")}
					output := &guardedOutput{t: t, checked: &checked}
					reports := 0
					streams := pluginsdk.ExecutionStreams{Report: func(pluginsdk.ProgressEvent) error {
						if !checked {
							t.Fatal("reported before compatibility check")
						}
						reports++
						return nil
					}}
					if command == "import" {
						r.Args = []string{"named"}
						streams.Input = input
					} else {
						streams.Output = output
					}
					result, err := requireStreaming(t, p).ExecuteStream(t.Context(), r, streams)
					if !target.allowed {
						if err == nil || calls != 1 || input.reads != 0 || output.writes != 0 || reports != 0 {
							t.Fatalf("calls %d reads %d writes %d reports %d: %v", calls, input.reads, output.writes, reports, err)
						}
						return
					}
					if err != nil || calls != 2 || !json.Valid(result.Data) {
						t.Fatalf("result %s calls %d: %v", result.Data, calls, err)
					}
					if command == "export" && output.String() != "{\"id\":1}" {
						t.Fatal("export was rendered instead of streamed")
					}
				})
			}
		}
	}
}

//nolint:gocognit // Keep fallback and boundary validation beside their HTTP call counts.
func TestStreamWrappersUnaryFallbackAndValidation(t *testing.T) {
	for _, bundle := range []bool{false, true} {
		t.Run(fmt.Sprint(bundle), func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: streamRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path == "/_info" {
					return streamHTTPResponse(200, `{"version":"2.5.1"}`), nil
				}
				return streamHTTPResponse(200, `{"data":[]}`), nil
			})}
			p := NewService("auth", client)
			path := []string{"auth", "clients", "list"}
			if bundle {
				p = New(client)
				path = append([]string{"legacy"}, path...)
			}
			r := pluginsdk.ExecuteRequest{Endpoint: "https://auth.example", CommandPath: path}
			result, err := requireStreaming(t, p).ExecuteStream(t.Context(), r, pluginsdk.ExecutionStreams{})
			if err != nil || calls != 2 || string(result.Data) != `{"data":[]}` {
				t.Fatalf("fallback %s calls %d: %v", result.Data, calls, err)
			}
			r.Flags = map[string]string{"unknown": "true"}
			_, err = requireStreaming(t, p).ExecuteStream(t.Context(), r, pluginsdk.ExecutionStreams{Input: strings.NewReader("must not read")})
			if err == nil || calls != 2 {
				t.Fatalf("invalid request made HTTP call: %v", err)
			}
		})
	}
	_, err := requireStreaming(t, NewService("unknown", nil)).ExecuteStream(t.Context(), pluginsdk.ExecuteRequest{}, pluginsdk.ExecutionStreams{})
	if err == nil {
		t.Fatal("unknown service accepted")
	}
}

//nolint:gocognit // Walk both manifests and assert exact metadata for the only streaming commands.
func TestStreamManifestOnlyLedgerImportExport(t *testing.T) {
	for _, p := range []pluginsdk.Plugin{New(nil), NewService("ledger", nil)} {
		m, err := p.GetManifest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if m.ProtocolVersion != 1 {
			t.Fatal("protocol version changed")
		}
		count := 0
		var walk func(pluginsdk.CommandSpec)
		walk = func(c pluginsdk.CommandSpec) {
			if c.Streams != nil {
				count++
				switch pluginsdk.CommandName(c) {
				case "import":
					if !reflect.DeepEqual(c.Streams, &pluginsdk.StreamSpec{Input: true}) || !reflect.DeepEqual(c.Files, &pluginsdk.FileSpec{ReadArgument: new(1), ReadFlag: "file", ReadFormat: "string"}) {
						t.Fatalf("import metadata %+v %+v", c.Streams, c.Files)
					}
				case "export":
					if !reflect.DeepEqual(c.Streams, &pluginsdk.StreamSpec{Output: true}) || !reflect.DeepEqual(c.Files, &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}) {
						t.Fatalf("export metadata %+v %+v", c.Streams, c.Files)
					}
				default:
					t.Fatalf("unexpected stream command %s", c.Use)
				}
			}
			for _, child := range c.Subcommands {
				walk(child)
			}
		}
		walk(m.Root)
		if count != 2 {
			t.Fatalf("%d stream commands", count)
		}
	}
}

func TestStreamWrappersInfoFailureDoesNotRead(t *testing.T) {
	failure := errors.New("info unavailable")
	client := &http.Client{Transport: streamRoundTripper(func(*http.Request) (*http.Response, error) { return nil, failure })}
	checked := false
	input := &guardedInput{t: t, checked: &checked, reader: strings.NewReader("{\"id\":1}")}
	_, err := requireStreaming(t, NewService("ledger", client)).ExecuteStream(t.Context(), pluginsdk.ExecuteRequest{Endpoint: "https://ledger.example", CommandPath: []string{"ledger", "import"}, Args: []string{"named"}}, pluginsdk.ExecutionStreams{Input: input})
	if !errors.Is(err, failure) || input.reads != 0 {
		t.Fatalf("read %d: %v", input.reads, err)
	}
}
