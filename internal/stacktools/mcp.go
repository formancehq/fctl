package stacktools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

type MCPOptions struct {
	Endpoint string
	Client   *http.Client
	Timeout  time.Duration
	Input    io.Reader
	Output   io.Writer
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpRemote struct {
	endpoint  string
	client    *http.Client
	sessionID string
	version   string
}

type messageRead struct {
	data []byte
	err  error
}

// ServeMCP bridges sequential stdio requests to the stack's HTTP MCP endpoint.
func ServeMCP(ctx context.Context, options MCPOptions) error {
	target, err := endpoint(options.Endpoint)
	if err != nil {
		return err
	}
	client, err := requestClient(options.Client, options.Timeout)
	if err != nil {
		return err
	}
	if options.Input == nil || options.Output == nil {
		return fmt.Errorf("MCP input and output are required")
	}
	target.Path = strings.TrimRight(target.Path, "/") + "/api/mcp"
	remote := &mcpRemote{endpoint: target.String(), client: client, version: "2024-11-05"}
	err = remote.serve(ctx, options.Input, options.Output)
	if ctx.Err() != nil {
		if closer, ok := options.Input.(io.Closer); ok {
			return errors.Join(err, closer.Close())
		}
	}
	return err
}

func (remote *mcpRemote) serve(ctx context.Context, input io.Reader, output io.Writer) error {
	reads := make(chan messageRead)
	done := make(chan struct{})
	defer close(done)
	go readMessages(input, reads, done)
	for {
		data, err := nextMessage(ctx, reads)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := remote.handle(ctx, data, output); err != nil {
			return err
		}
	}
}

func nextMessage(ctx context.Context, reads <-chan messageRead) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case read := <-reads:
		return read.data, read.err
	}
}

func readMessages(input io.Reader, reads chan<- messageRead, done <-chan struct{}) {
	reader := bufio.NewReader(input)
	for {
		data, err := readMessage(reader)
		select {
		case reads <- messageRead{data: data, err: err}:
		case <-done:
			return
		}
		if err != nil {
			return
		}
	}
}

func (remote *mcpRemote) handle(ctx context.Context, data []byte, output io.Writer) error {
	var message rpcMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return writeRPCError(output, json.RawMessage("null"), -32700, "parse error")
	}
	if message.JSONRPC != "2.0" || message.Method == "" || !validID(message.ID) {
		return writeRPCError(output, json.RawMessage("null"), -32600, "invalid request")
	}
	if message.Method == "ping" && len(message.ID) > 0 {
		response := struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  struct{}        `json:"result"`
		}{JSONRPC: "2.0", ID: message.ID}
		return json.NewEncoder(output).Encode(response)
	}
	response, err := remote.send(ctx, message, data)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(message.ID) == 0 {
			return err
		}
		return writeRPCError(output, message.ID, -32000, err.Error())
	}
	if len(message.ID) == 0 {
		return nil
	}
	return json.NewEncoder(output).Encode(response)
}

func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return false
	}
	switch value.(type) {
	case nil, string, json.Number:
		return true
	default:
		return false
	}
}

func writeRPCError(out io.Writer, id json.RawMessage, code int, message string) error {
	response := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{JSONRPC: "2.0", ID: id}
	response.Error.Code, response.Error.Message = code, message
	return json.NewEncoder(out).Encode(response)
}

func (remote *mcpRemote) send(ctx context.Context, message rpcMessage, data []byte) (result json.RawMessage, sendErr error) {
	request, err := remote.request(ctx, message, data)
	if err != nil {
		return nil, err
	}
	response, err := remote.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("MCP request failed")
	}
	defer func() { sendErr = errors.Join(sendErr, response.Body.Close()) }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("MCP HTTP %d", response.StatusCode)
	}
	if id := response.Header.Get("Mcp-Session-Id"); id != "" {
		remote.sessionID = id
	}
	if len(message.ID) == 0 {
		return nil, nil
	}
	result, err = decodeResponse(response, message.ID)
	if err == nil && message.Method == "initialize" {
		remote.negotiatedVersion(result)
	}
	return result, err
}

func (remote *mcpRemote) request(ctx context.Context, message rpcMessage, data []byte) (*http.Request, error) {
	if message.Method == "initialize" {
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid initialize parameters")
		}
		if params.ProtocolVersion != "" {
			remote.version = params.ProtocolVersion
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, remote.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot build MCP request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", remote.version)
	if remote.sessionID != "" {
		request.Header.Set("Mcp-Session-Id", remote.sessionID)
	}
	return request, nil
}

func (remote *mcpRemote) negotiatedVersion(payload json.RawMessage) {
	var response struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if json.Unmarshal(payload, &response) == nil && response.Result.ProtocolVersion != "" {
		remote.version = response.Result.ProtocolVersion
	}
}

func decodeResponse(response *http.Response, id json.RawMessage) (json.RawMessage, error) {
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("invalid MCP response content type")
	}
	if mediaType == "text/event-stream" {
		return readSSE(response.Body, id)
	}
	if mediaType != "application/json" {
		return nil, fmt.Errorf("unsupported MCP response content type")
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, MaxMessageSize+1))
	if err != nil {
		return nil, fmt.Errorf("MCP response read failed")
	}
	if len(payload) > MaxMessageSize {
		return nil, fmt.Errorf("MCP response exceeds size limit")
	}
	if !matchesResponse(payload, id) {
		return nil, fmt.Errorf("MCP response ID does not match request")
	}
	return payload, nil
}

func matchesResponse(payload, id json.RawMessage) bool {
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(payload, &response) != nil || response.JSONRPC != "2.0" {
		return false
	}
	if (len(response.Result) == 0) == (len(response.Error) == 0) {
		return false
	}
	var a, b bytes.Buffer
	if json.Compact(&a, response.ID) != nil || json.Compact(&b, id) != nil {
		return false
	}
	return bytes.Equal(a.Bytes(), b.Bytes())
}
