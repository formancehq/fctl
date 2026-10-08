package stacktools_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/internal/stacktools"
)

func TestMCPHTTPAndSSESession(t *testing.T) {
	t.Parallel()
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		handleMCPFixture(t, w, r)
	}))
	t.Cleanup(server.Close)
	input := `{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26"}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":9007199254740993,"method":"tools/call","params":{"name":"list_ledgers"}}
{"jsonrpc":"2.0","id":"fail","method":"tools/call"}
{"jsonrpc":"2.0","id":4,"method":"ping"}
`
	var out bytes.Buffer
	err := stacktools.ServeMCP(t.Context(), stacktools.MCPOptions{Endpoint: server.URL + "/gateway", Client: authenticatedClient(server), Timeout: time.Second, Input: strings.NewReader(input), Output: &out})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	want := []string{`{"jsonrpc":"2.0","id":"init","result":{"protocolVersion":"2025-03-26","capabilities":{}}}`, `{"jsonrpc":"2.0","id":9007199254740993,"result":{"content":[]}}`, `{"jsonrpc":"2.0","id":"fail","error":{"code":-32602,"message":"invalid params","data":{"reason":"fixture"}}}`, `{"jsonrpc":"2.0","id":4,"result":{}}`}
	if len(lines) != len(want) {
		t.Fatalf("protocol output %s", out.String())
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("response %d = %s, want %s", i, lines[i], want[i])
		}
	}
	if count.Load() != 4 {
		t.Fatalf("remote requests %d", count.Load())
	}
}

func handleMCPFixture(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.URL.Path != "/gateway/api/mcp" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer synthetic-stack-token" {
		t.Error("incorrect MCP route/auth")
	}
	if r.Header.Get("MCP-Protocol-Version") != "2025-03-26" || r.Header.Get("Accept") != "application/json, text/event-stream" {
		t.Error("protocol negotiation missing")
	}
	var message struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if message.Method == "initialize" {
		w.Header().Set("Mcp-Session-Id", "fixture-session")
		w.Header().Set("Content-Type", "application/json")
		writeFixture(t, w, `{"jsonrpc":"2.0","id":"init","result":{"protocolVersion":"2025-03-26","capabilities":{}}}`)
		return
	}
	if r.Header.Get("Mcp-Session-Id") != "fixture-session" {
		t.Error("MCP session missing")
	}
	if len(message.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	if string(message.ID) == `"fail"` {
		writeFixture(t, w, "data: {\"jsonrpc\":\"2.0\",\"id\":\"fail\",\"error\":{\"code\":-32602,\"message\":\"invalid params\",\"data\":{\"reason\":\"fixture\"}}}\n\n")
		return
	}
	writeFixture(t, w, ": keepalive\n\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\ndata: {\"jsonrpc\":\"2.0\",\"id\":9007199254740993,\"result\":{\"content\":[]}}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	<-r.Context().Done()
}

func writeFixture(t *testing.T, w io.Writer, data string) {
	t.Helper()
	if _, err := io.WriteString(w, data); err != nil {
		t.Error(err)
	}
}

func TestMCPFramingAndValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input string
		want        string
		failure     bool
	}{
		{name: "content length", input: fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), `{"jsonrpc":"2.0","id":1,"method":"ping"}`), want: `"id":1`},
		{name: "parse error", input: "not json\n", want: `"code":-32700`},
		{name: "invalid id", input: `{"jsonrpc":"2.0","id":{},"method":"ping"}`, want: `"code":-32600`},
		{name: "truncated frame", input: "Content-Length: 10\r\n\r\n", failure: true},
		{name: "truncated headers", input: "Content-Length: 10\r\n", failure: true},
		{name: "oversized frame", input: fmt.Sprintf("Content-Length: %d\r\n\r\n", stacktools.MaxMessageSize+1), failure: true},
		{name: "oversized line", input: strings.Repeat("x", stacktools.MaxMessageSize+1), failure: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			err := stacktools.ServeMCP(t.Context(), stacktools.MCPOptions{Endpoint: "http://127.0.0.1", Client: &http.Client{}, Timeout: time.Second, Input: strings.NewReader(test.input), Output: &out})
			if test.failure {
				if err == nil {
					t.Fatal("invalid framing accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), test.want) {
				t.Fatalf("output %s", out.String())
			}
		})
	}
}

func TestMCPRejectsMismatchedIDAndRedirect(t *testing.T) {
	t.Parallel()
	var external atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { external.Add(1) }))
	t.Cleanup(other.Close)
	tests := []struct {
		name, body, contentType string
		status                  int
	}{
		{name: "wrong id", body: `{"jsonrpc":"2.0","id":2,"result":{}}`, contentType: "application/json", status: http.StatusOK},
		{name: "wrong SSE id", body: "data: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{}}\n\n", contentType: "text/event-stream", status: http.StatusOK},
		{name: "redirect", status: http.StatusTemporaryRedirect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", other.URL)
				w.Header().Set("Content-Type", test.contentType)
				w.WriteHeader(test.status)
				writeFixture(t, w, test.body)
			}))
			t.Cleanup(server.Close)
			var out bytes.Buffer
			err := stacktools.ServeMCP(t.Context(), stacktools.MCPOptions{Endpoint: server.URL, Client: authenticatedClient(server), Timeout: time.Second, Input: strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), Output: &out})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `"id":1,"error":{"code":-32000`) || strings.Contains(out.String(), "synthetic-stack-token") {
				t.Fatalf("unsafe response %s", out.String())
			}
		})
	}
	if external.Load() != 0 {
		t.Fatal("MCP bearer followed redirect")
	}
}

func TestMCPCancellation(t *testing.T) {
	t.Parallel()
	input, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- stacktools.ServeMCP(ctx, stacktools.MCPOptions{Endpoint: "http://127.0.0.1", Client: &http.Client{}, Timeout: time.Second, Input: input, Output: io.Discard})
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("MCP did not cancel blocked input")
	}
}

func TestMCPNegotiatedVersionAndMultilineSSE(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Error(err)
			return
		}
		if message.Method == "initialize" {
			w.Header().Set("Content-Type", "application/json")
			writeFixture(t, w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}`)
			return
		}
		if r.Header.Get("MCP-Protocol-Version") != "2024-11-05" {
			t.Error("negotiated protocol ignored")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		writeFixture(t, w, "event: message\r\ndata: {\"jsonrpc\":\"2.0\",\r\ndata: \"id\":2,\"result\":{\"tools\":[]}}\r\n\r\n")
	}))
	t.Cleanup(server.Close)
	var out bytes.Buffer
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-03-26\"}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n"
	err := stacktools.ServeMCP(t.Context(), stacktools.MCPOptions{Endpoint: server.URL, Client: server.Client(), Timeout: time.Second, Input: strings.NewReader(input), Output: &out})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id":2,"result":{"tools":[]}`) {
		t.Fatalf("multiline SSE %s", out.String())
	}
}
