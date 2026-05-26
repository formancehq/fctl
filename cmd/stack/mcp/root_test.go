package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestStdioServerForwardsInitialize(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mcp" {
			t.Fatalf("path = %s, want /api/mcp", r.URL.Path)
		}
		var request rpcMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decoding remote request: %v", err)
		}
		if request.Method != "initialize" {
			t.Fatalf("method = %s, want initialize", request.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"remote"}}}`))
	}))
	defer upstream.Close()

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	var output bytes.Buffer
	server := &stdioServer{
		in:         strings.NewReader(input),
		out:        &output,
		err:        &bytes.Buffer{},
		httpClient: upstream.Client(),
		stackURI:   upstream.URL,
	}

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response rpcResponse
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if string(response.ID) != "1" {
		t.Fatalf("response id = %s, want 1", response.ID)
	}
	if response.Error != nil {
		t.Fatalf("response error = %#v", response.Error)
	}
}

func TestReadMCPMessageContentLength(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":"abc","method":"ping"}`)
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)

	got, err := readMCPMessage(bufioReader(input))
	if err != nil {
		t.Fatalf("readMCPMessage() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestReadMCPMessageRejectsNegativeContentLength(t *testing.T) {
	_, err := readMCPMessage(bufioReader("Content-Length: -1\r\n\r\n"))
	if err == nil {
		t.Fatalf("readMCPMessage() expected error")
	}
	if !strings.Contains(err.Error(), "must be non-negative") {
		t.Fatalf("readMCPMessage() error = %v, want non-negative framing error", err)
	}
}

func TestStdioServerForwardsCancelledNotification(t *testing.T) {
	var gotMethod atomic.Value
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request rpcMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decoding remote request: %v", err)
		}
		gotMethod.Store(request.Method)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()

	input := `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}` + "\n"
	server := &stdioServer{
		in:         strings.NewReader(input),
		out:        &bytes.Buffer{},
		err:        &bytes.Buffer{},
		httpClient: upstream.Client(),
		stackURI:   upstream.URL,
	}

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if got := gotMethod.Load(); got != "notifications/cancelled" {
		t.Fatalf("forwarded method = %v, want notifications/cancelled", got)
	}
}

func TestStdioServerPreservesForwardedMethodErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
	}))
	defer upstream.Close()

	input := `{"jsonrpc":"2.0","id":1,"method":"resources/list"}` + "\n"
	var output bytes.Buffer
	server := &stdioServer{
		in:         strings.NewReader(input),
		out:        &output,
		err:        &bytes.Buffer{},
		httpClient: upstream.Client(),
		stackURI:   upstream.URL,
	}

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response rpcResponse
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if response.Error == nil {
		t.Fatalf("response error is nil")
	}
	if response.Error.Code != -32000 {
		t.Fatalf("response error code = %d, want -32000", response.Error.Code)
	}
	if !strings.Contains(response.Error.Message, "remote MCP HTTP 401") {
		t.Fatalf("response error message = %q, want remote HTTP error", response.Error.Message)
	}
}

func TestRemoteMCPClientKeepsSessionID(t *testing.T) {
	var count atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber := count.Add(1)
		if requestNumber == 1 {
			w.Header().Set("Mcp-Session-Id", "session-123")
		} else if got := r.Header.Get("Mcp-Session-Id"); got != "session-123" {
			t.Fatalf("Mcp-Session-Id = %q, want session-123", got)
		}
		if got := r.Header.Get("MCP-Protocol-Version"); got != "2025-03-26" {
			t.Fatalf("MCP-Protocol-Version = %q, want 2025-03-26", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	defer upstream.Close()

	client := newRemoteMCPClient(upstream.Client(), upstream.URL)
	_, err := client.Request(context.Background(), rpcMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-03-26"}`),
	})
	if err != nil {
		t.Fatalf("initialize request error = %v", err)
	}

	_, err = client.Request(context.Background(), rpcMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/list",
	})
	if err != nil {
		t.Fatalf("tools/list request error = %v", err)
	}
}

func TestDecodeSSEResponse(t *testing.T) {
	resp, err := decodeRemoteMCPResponse("text/event-stream", []byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"ok\":true}}\n\n"))
	if err != nil {
		t.Fatalf("decodeRemoteMCPResponse() error = %v", err)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("result = %#v, want ok=true", resp.Result)
	}
}

func bufioReader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}
