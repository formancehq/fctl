package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

type fakePlugin struct {
	manifest    pluginsdk.Manifest
	manifestErr error
	execute     func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error)
}

func (f *fakePlugin) GetManifest(context.Context) (pluginsdk.Manifest, error) {
	return f.manifest, f.manifestErr
}
func (f *fakePlugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	return f.execute(ctx, req)
}

func testManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "ledger", Version: "test", Service: "ledger", ProtocolVersion: 1, Root: pluginsdk.CommandSpec{
		Use: "ledger", Short: "Ledger fixture", Flags: []pluginsdk.FlagSpec{{Name: "tenant", Type: "string", Required: true, Persistent: true}, {Name: "enabled", Type: "bool", Default: "true", Persistent: true}},
		Subcommands: []pluginsdk.CommandSpec{{Use: "write NAME", Runnable: true, Confirm: true, Args: pluginsdk.ArgsSpec{Min: 1, Max: 1}, Flags: []pluginsdk.FlagSpec{{Name: "confirm", Type: "bool", Default: "false"}, {Name: "data", Type: "string", Body: true}, {Name: "limit", Type: "uint32", Default: "10"}}}},
	}}
}

func attach(t *testing.T, m pluginsdk.Manifest, resolve plugin.Resolver, factory plugin.Factory) *cobra.Command {
	t.Helper()
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, factory); err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "fctl", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().String("profile", "", "host setting")
	if err := plugin.NewCommand(registry, resolve).AddTo(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAdapterBodyAndAuthenticatedClient(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"inline", "file", "stdin"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); testBodyCase(t, kind) })
	}
}

func bodyInput(t *testing.T, kind, body string) string {
	t.Helper()
	switch kind {
	case "stdin":
		return "-"
	case "file":
		path := filepath.Join(t.TempDir(), "body.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return "@" + path
	default:
		return body
	}
}

func bodyServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prefix/api/ledger/invoke" || r.Method != http.MethodPost {
			t.Error("wrong service route")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("injected authentication missing")
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if string(data) != body {
			t.Error("request JSON number changed")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func recordingFactory(t *testing.T, expected *http.Client, got *pluginsdk.ExecuteRequest) plugin.Factory {
	t.Helper()
	return func(injected *http.Client) pluginsdk.Plugin {
		if injected != expected {
			t.Error("factory did not receive resolved HTTP client")
		}
		return &fakePlugin{execute: func(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			*got = req
			client, err := api.New(req.Endpoint, injected)
			if err != nil {
				return pluginsdk.ExecuteResponse{}, err
			}
			data, err := client.Do(ctx, http.MethodPost, "/invoke", nil, req.Body, nil)
			return pluginsdk.ExecuteResponse{Data: data}, err
		}}
	}
}

func testBodyCase(t *testing.T, kind string) {
	t.Helper()
	body := `{"amount":9007199254740993}`
	value := bodyInput(t, kind, body)
	server := bodyServer(t, body)
	httpClient := &http.Client{Transport: bearerTransport{server.Client().Transport}}
	calls := 0
	var got pluginsdk.ExecuteRequest
	endpoint := server.URL + "/prefix/api/ledger"
	root := attach(t, testManifest(), func(_ context.Context, service string) (*api.Client, error) {
		calls++
		if service != "ledger" {
			t.Error("wrong service")
		}
		return api.New(endpoint, httpClient)
	}, recordingFactory(t, httpClient, &got))
	root.SetIn(strings.NewReader(body))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--profile", "host", "ledger", "--tenant", "org", "write", "demo", "--confirm", "--enabled=false", "--limit", "42", "--data", value})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || got.Endpoint != endpoint || string(got.Body) != body {
		t.Fatal("wrong resolved request")
	}
	assertBodyRequest(t, got, value)
	if out.String() != "{\n  \"amount\": 9007199254740993\n}\n" {
		t.Fatalf("JSON output = %q", out.String())
	}
}

func assertBodyRequest(t *testing.T, got pluginsdk.ExecuteRequest, value string) {
	t.Helper()
	if !reflect.DeepEqual(got.CommandPath, []string{"ledger", "write"}) || !reflect.DeepEqual(got.Args, []string{"demo"}) {
		t.Fatal("wrong command path or positional arguments")
	}
	wantFlags := map[string]string{"tenant": "org", "enabled": "false", "confirm": "true", "limit": "42", "data": value}
	if !reflect.DeepEqual(got.Flags, wantFlags) {
		t.Fatalf("plugin flags = %v", got.Flags)
	}
	for name := range wantFlags {
		if !got.ChangedFlags[name] {
			t.Errorf("missing changed flag %s", name)
		}
	}
}

type bearerTransport struct{ base http.RoundTripper }

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer test-token")
	return b.base.RoundTrip(req)
}

func TestAdapterValidationBeforeResolution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing confirm", []string{"ledger", "--tenant", "org", "write", "demo", "--data", "-"}},
		{"missing required persistent flag", []string{"ledger", "write", "demo", "--confirm"}},
		{"missing arg", []string{"ledger", "--tenant", "org", "write", "--confirm"}},
		{"extra arg", []string{"ledger", "--tenant", "org", "write", "one", "two", "--confirm"}},
		{"invalid uint32", []string{"ledger", "--tenant", "org", "write", "demo", "--confirm", "--limit", "4294967296"}},
		{"invalid bool", []string{"ledger", "--tenant", "org", "write", "demo", "--confirm=invalid"}},
		{"explicit empty body", []string{"ledger", "--tenant", "org", "write", "demo", "--confirm", "--data="}},
		{"invalid JSON", []string{"ledger", "--tenant", "org", "write", "demo", "--confirm", "--data", "{"}},
		{"missing file", []string{"ledger", "--tenant", "org", "write", "demo", "--confirm", "--data", "@/nonexistent-fctl-plugin-body"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := attach(t, testManifest(), func(context.Context, string) (*api.Client, error) {
				t.Error("invalid command resolved HTTP client")
				return nil, errors.New("unexpected resolver")
			}, func(*http.Client) pluginsdk.Plugin { t.Error("invalid command constructed plugin"); return nil })
			root.SetIn(forbiddenReader{t})
			root.SetArgs(tc.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.ExecuteContext(t.Context()); err == nil {
				t.Fatal("invalid command accepted")
			}
		})
	}
}

type forbiddenReader struct{ t *testing.T }

func (r forbiddenReader) Read([]byte) (int, error) {
	r.t.Error("body read before command validation")
	return 0, io.EOF
}

func TestAdapterHelpWithoutHTTP(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"ledger", "--help"}, {"ledger", "write", "--help"}, {"__complete", "ledger", ""}} {
		root := attach(t, testManifest(), func(context.Context, string) (*api.Client, error) {
			t.Error("help resolved HTTP client")
			return nil, nil
		}, func(*http.Client) pluginsdk.Plugin { t.Error("help constructed plugin"); return nil })
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(io.Discard)
		root.SetArgs(args)
		if err := root.ExecuteContext(t.Context()); err != nil {
			t.Fatal(err)
		}
		if out.Len() == 0 {
			t.Fatal("help or completion output missing")
		}
	}
}

func TestAdapterDefaultsAndPartialError(t *testing.T) {
	t.Parallel()
	executionErr := errors.New("bulk partially failed")
	writerErr := errors.New("stdout failed")
	for _, failWrite := range []bool{false, true} {
		m := testManifest()
		// Exercise MaxArgs and an inherited body with a nonempty default.
		m.Root.Flags = append(m.Root.Flags, pluginsdk.FlagSpec{Name: "payload", Type: "string", Default: `{"x":1}`, Body: true, Persistent: true})
		m.Root.Subcommands[0].Flags = m.Root.Subcommands[0].Flags[:1]
		m.Root.Subcommands[0].Args = pluginsdk.ArgsSpec{Max: 2}
		root := attach(t, m, func(context.Context, string) (*api.Client, error) {
			return api.New("http://localhost:9000", http.DefaultClient)
		}, func(*http.Client) pluginsdk.Plugin {
			return &fakePlugin{execute: func(_ context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
				if req.Flags["enabled"] != "true" || req.ChangedFlags["enabled"] || req.ChangedFlags["payload"] {
					t.Error("unchanged defaults marked as changed")
				}
				if string(req.Body) != `{"x":1}` {
					t.Error("inherited default body not read")
				}
				return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"ok":1,"failed":1}`)}, executionErr
			}}
		})
		var out bytes.Buffer
		root.SetOut(&out)
		if failWrite {
			root.SetOut(errorWriter{writerErr})
		}
		root.SetArgs([]string{"ledger", "write", "--tenant", "org", "--confirm"})
		err := root.ExecuteContext(t.Context())
		assertPartialResult(t, err, executionErr, writerErr, failWrite, out.String())
	}
}

func assertPartialResult(t *testing.T, err, executionErr, writerErr error, failWrite bool, out string) {
	t.Helper()
	if !errors.Is(err, executionErr) {
		t.Fatalf("lost execution error: %v", err)
	}
	if failWrite && !errors.Is(err, writerErr) {
		t.Fatalf("lost writer error: %v", err)
	}
	if !failWrite && out != "{\n  \"ok\": 1,\n  \"failed\": 1\n}\n" {
		t.Fatalf("partial result missing: %q", out)
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestAdapterPassesIsolatedHostContext(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		metadata map[string]string
	}{
		{"default nil", nil},
		{"host metadata", map[string]string{"organization": "org-1", "stack": "stack-1", "organizationIDs": `["org-1","org-2"]`}},
	} {
		t.Run(tc.name, func(t *testing.T) { testAdapterContext(t, tc.metadata) })
	}
}

func testAdapterContext(t *testing.T, metadata map[string]string) {
	t.Helper()
	want := maps.Clone(metadata)
	httpClient := &http.Client{}
	base, err := api.New("https://ledger.example/prefix", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	resolved := base.WithContext(metadata)
	var received map[string]string
	root := attach(t, testManifest(), func(context.Context, string) (*api.Client, error) {
		return resolved, nil
	}, func(injected *http.Client) pluginsdk.Plugin {
		if injected != httpClient {
			t.Error("metadata replaced the injected HTTP client")
		}
		return &fakePlugin{execute: func(_ context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			received = req.Context
			return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"ok":true}`)}, nil
		}}
	})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"ledger", "--tenant", "cli-tenant", "write", "demo", "--confirm"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(received, want) || out.String() != "{\n  \"ok\": true\n}\n" {
		t.Fatal("adapter lost host context or printed it as command output")
	}
	if received != nil {
		received["organization"] = "plugin-change"
	}
	if !reflect.DeepEqual(resolved.Context(), want) || !reflect.DeepEqual(metadata, want) {
		t.Fatal("plugin mutation reached the resolver client or host metadata")
	}
}
