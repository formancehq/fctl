package cloudtools

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func utilityRoot(t *testing.T, resolve resolver, read tokenReader) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "fctl", SilenceUsage: true, SilenceErrors: true}
	settings := &connection.Settings{}
	settings.Bind(root)
	cloudCommand := &cobra.Command{Use: "cloud"}
	cloudCommand.AddCommand(&cobra.Command{Use: "stack"})
	root.AddCommand(cloudCommand)
	if err := addTo(root, resolve, read); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestHostedHelpAndValidationDoNotAuthenticate(t *testing.T) {
	t.Parallel()
	tests := [][]string{
		{"cloud", "stack", "proxy", "--help"},
		{"cloud", "stack", "mcp", "serve", "--help"},
		{"cloud", "generate-personal-token", "--help"},
		{"cloud", "stack", "proxy", "--port", "65536"},
		{"cloud", "stack", "proxy", "--timeout", "0s"},
		{"cloud", "stack", "mcp", "serve", "--timeout", "0s"},
		{"cloud", "stack", "mcp", "serve", "--transport", "http"},
		{"cloud", "generate-personal-token", "extra"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := utilityRoot(t, func(context.Context, *cobra.Command, string) (*api.Client, error) {
				t.Error("unexpected authentication")
				return nil, errors.New("unexpected authentication")
			}, func(context.Context, *http.Client) (string, error) {
				t.Error("unexpected token access")
				return "", nil
			})
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(args)
			err := root.ExecuteContext(t.Context())
			help := args[len(args)-1] == "--help"
			if help && err != nil {
				t.Fatal(err)
			}
			if !help && err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestHostedTokenIsExplicitJSON(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{}
	client, err := api.New("http://127.0.0.1", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	root := utilityRoot(t, func(_ context.Context, _ *cobra.Command, service string) (*api.Client, error) {
		if service != "stack" {
			t.Error("incorrect service")
		}
		return client, nil
	}, func(_ context.Context, got *http.Client) (string, error) {
		if got != httpClient {
			t.Error("authenticated client replaced")
		}
		return "synthetic-explicit-token", nil
	})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--output", "table", "cloud", "generate-personal-token"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\n  \"token\": \"synthetic-explicit-token\"\n}\n" || stderr.Len() != 0 {
		t.Fatalf("token output %q stderr %q", out.String(), stderr.String())
	}
}

func TestHostedMCPUsesCommandStreams(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prefix/api/mcp" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"jsonrpc":"2.0","id":"tool","result":{"tools":[]}}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL+"/prefix", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	root := utilityRoot(t, func(context.Context, *cobra.Command, string) (*api.Client, error) { return client, nil }, nil)
	root.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":"tool","method":"tools/list"}`))
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"cloud", "stack", "mcp", "serve"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\"jsonrpc\":\"2.0\",\"id\":\"tool\",\"result\":{\"tools\":[]}}\n" || stderr.Len() != 0 {
		t.Fatalf("protocol output %q stderr %q", out.String(), stderr.String())
	}
}

func TestAttachmentIsAtomic(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "fctl"}
	cloudCommand := &cobra.Command{Use: "cloud"}
	stack := &cobra.Command{Use: "stack"}
	stack.AddCommand(&cobra.Command{Use: "proxy"})
	cloudCommand.AddCommand(stack)
	root.AddCommand(cloudCommand)
	if err := addTo(root, nil, nil); err == nil {
		t.Fatal("collision accepted")
	}
	if child(stack, "mcp") != nil || child(cloudCommand, "generate-personal-token") != nil {
		t.Fatal("partial attachment")
	}
	if err := AddTo(root, nil); err == nil {
		t.Fatal("missing settings accepted")
	}
	if err := AddTo(&cobra.Command{Use: "fctl"}, &connection.Settings{Timeout: time.Second}); err == nil {
		t.Fatal("missing groups accepted")
	}
}

type failedWriter struct{ err error }

func (w failedWriter) Write([]byte) (int, error) { return 0, w.err }

func TestTokenWriterFailure(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("writer failed")
	client, err := api.New("http://127.0.0.1", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	root := utilityRoot(t, func(context.Context, *cobra.Command, string) (*api.Client, error) { return client, nil }, func(context.Context, *http.Client) (string, error) { return "synthetic-explicit-token", nil })
	root.SetOut(failedWriter{err: sentinel})
	root.SetArgs([]string{"cloud", "generate-personal-token"})
	if err := root.ExecuteContext(t.Context()); !errors.Is(err, sentinel) {
		t.Fatalf("writer error %v", err)
	}
}

type proxyReadyWriter struct{ ready chan<- string }

func (writer proxyReadyWriter) Write(data []byte) (int, error) {
	address := strings.TrimSpace(strings.TrimPrefix(string(data), "Stack proxy listening on http://"))
	writer.ready <- address
	return len(data), nil
}

func TestHostedProxyUsesStackClientAndCancels(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/api/ledger/v3/ledgers" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL+"/base", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	root := utilityRoot(t, func(_ context.Context, _ *cobra.Command, service string) (*api.Client, error) {
		if service != "stack" {
			t.Error("wrong proxy service")
		}
		return client, nil
	}, nil)
	ready := make(chan string, 1)
	root.SetErr(proxyReadyWriter{ready: ready})
	root.SetOut(io.Discard)
	root.SetArgs([]string{"cloud", "stack", "proxy", "--port", "0"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()
	assertHostedProxyRequest(t, ready)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("command did not stop")
	}
}

func assertHostedProxyRequest(t *testing.T, ready <-chan string) {
	t.Helper()
	var address string
	select {
	case address = <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy command did not start")
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address+"/api/ledger/v3/ledgers", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&http.Client{Timeout: time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("proxy status %d", response.StatusCode)
	}
}
