package mcp

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
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	transportFlag     = "transport"
	maxMCPMessageSize = 10 * 1024 * 1024
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("mcp",
		fctl.WithShortDescription("Run stack MCP integrations"),
		fctl.WithChildCommands(NewServeCommand()),
	)
}

func NewServeCommand() *cobra.Command {
	return fctl.NewStackCommand("serve",
		fctl.WithShortDescription("Start a stack MCP server"),
		fctl.WithStringFlag(transportFlag, "stdio", "MCP transport to use (stdio)"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithRunE(runServe),
	)
}

func runServe(cmd *cobra.Command, _ []string) error {
	transport := fctl.GetString(cmd, transportFlag)
	if transport != "stdio" {
		return fmt.Errorf("unsupported MCP transport %q: only stdio is currently supported", transport)
	}

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return err
	}

	stackToken, stackAccess, err := fctl.EnsureStackAccess(cmd, relyingParty, stderrDialog{w: cmd.ErrOrStderr()}, profileName, *profile, organizationID, stackID)
	if err != nil {
		return err
	}

	tokenSource := fctl.NewStackTokenSource(
		*stackToken,
		stackAccess,
		relyingParty,
		func(newToken fctl.AccessToken) error {
			return fctl.WriteStackToken(cmd, profileName, stackID, newToken)
		},
		cmd,
		profileName,
		organizationID,
		stackID,
	)
	httpClient := oauth2.NewClient(cmd.Context(), tokenSource)

	server := &stdioServer{
		in:         os.Stdin,
		out:        os.Stdout,
		err:        cmd.ErrOrStderr(),
		httpClient: httpClient,
		stackURI:   stackAccess.URI,
	}
	return server.Serve(cmd.Context())
}

type stderrDialog struct {
	w io.Writer
}

func (d stderrDialog) Info(msg string, args ...any) {
	_, _ = fmt.Fprintf(d.w, msg+"\n", args...)
}

type stdioServer struct {
	in         io.Reader
	out        io.Writer
	err        io.Writer
	httpClient *http.Client
	stackURI   string
	remote     *remoteMCPClient
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *stdioServer) Serve(ctx context.Context) error {
	s.remote = newRemoteMCPClient(s.httpClient, s.stackURI)
	reader := bufio.NewReader(s.in)

	type readResult struct {
		data []byte
		err  error
	}
	reads := make(chan readResult, 1)
	go func() {
		for {
			data, err := readMCPMessage(reader)
			select {
			case reads <- readResult{data: data, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		var read readResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case read = <-reads:
		}

		data, err := read.data, read.err
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if len(bytes.TrimSpace(data)) == 0 {
			continue
		}

		var msg rpcMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = s.writeResponse(rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}

		if len(msg.ID) == 0 {
			s.handleNotification(msg)
			continue
		}

		result, rpcErr := s.handleRequest(ctx, msg)
		resp := rpcResponse{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  result,
			Error:   rpcErr,
		}
		if err := s.writeResponse(resp); err != nil {
			return err
		}
	}
}

func (s *stdioServer) handleNotification(msg rpcMessage) {
	switch msg.Method {
	case "notifications/cancelled":
		_, _ = fmt.Fprintf(s.err, "MCP request cancelled\n")
		if err := s.remote.Notify(context.Background(), msg); err != nil {
			_, _ = fmt.Fprintf(s.err, "forwarding MCP notification %q failed: %v\n", msg.Method, err)
		}
	case "notifications/initialized":
		if err := s.remote.Notify(context.Background(), msg); err != nil {
			_, _ = fmt.Fprintf(s.err, "forwarding MCP notification %q failed: %v\n", msg.Method, err)
		}
	}
}

func (s *stdioServer) handleRequest(ctx context.Context, msg rpcMessage) (any, *rpcError) {
	if msg.Method == "ping" {
		return map[string]any{}, nil
	}
	result, err := s.remote.Request(ctx, msg)
	if err != nil {
		return nil, &rpcError{Code: -32000, Message: err.Error()}
	}
	return result, nil
}

type remoteMCPClient struct {
	httpClient      *http.Client
	endpoint        string
	sessionID       string
	protocolVersion string
}

func newRemoteMCPClient(httpClient *http.Client, stackURI string) *remoteMCPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base, err := url.Parse(stackURI)
	if err != nil {
		return &remoteMCPClient{httpClient: httpClient, endpoint: stackURI, protocolVersion: "2024-11-05"}
	}
	endpoint := base.ResolveReference(&url.URL{Path: "/api/mcp"}).String()
	return &remoteMCPClient{httpClient: httpClient, endpoint: endpoint, protocolVersion: "2024-11-05"}
}

func (c *remoteMCPClient) Request(ctx context.Context, msg rpcMessage) (any, error) {
	resp, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("remote MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

func (c *remoteMCPClient) Notify(ctx context.Context, msg rpcMessage) error {
	_, err := c.send(ctx, msg)
	return err
}

func (c *remoteMCPClient) send(ctx context.Context, msg rpcMessage) (*rpcResponse, error) {
	if msg.Method == "initialize" {
		c.captureProtocolVersion(msg.Params)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", c.protocolVersion)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if sessionID := httpResp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.sessionID = sessionID
	}

	if httpResp.StatusCode == http.StatusAccepted && len(msg.ID) == 0 {
		return &rpcResponse{JSONRPC: "2.0"}, nil
	}

	payload, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote MCP HTTP %d: %s", httpResp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return &rpcResponse{JSONRPC: "2.0"}, nil
	}

	return decodeRemoteMCPResponse(httpResp.Header.Get("Content-Type"), payload)
}

func (c *remoteMCPClient) captureProtocolVersion(params json.RawMessage) {
	var initParams struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(params, &initParams); err == nil && initParams.ProtocolVersion != "" {
		c.protocolVersion = initParams.ProtocolVersion
	}
}

func decodeRemoteMCPResponse(contentType string, payload []byte) (*rpcResponse, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}

	if mediaType == "text/event-stream" {
		return decodeSSEResponse(payload)
	}

	var resp rpcResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, fmt.Errorf("decoding remote MCP response: %w", err)
	}
	return &resp, nil
}

func decodeSSEResponse(payload []byte) (*rpcResponse, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	var events [][]string
	var dataLines []string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if len(dataLines) > 0 {
				events = append(events, dataLines)
				dataLines = nil
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(dataLines) > 0 {
		events = append(events, dataLines)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("remote MCP SSE response did not contain data")
	}
	if len(events) > 1 {
		// The stdio bridge writes one JSON-RPC response per request; streaming SSE is not supported yet.
		return nil, fmt.Errorf("remote MCP SSE response contained multiple events")
	}
	var resp rpcResponse
	if err := json.Unmarshal([]byte(strings.Join(events[0], "\n")), &resp); err != nil {
		return nil, fmt.Errorf("decoding remote MCP SSE response: %w", err)
	}
	return &resp, nil
}

func readMCPMessage(reader *bufio.Reader) ([]byte, error) {
	for {
		first, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}
		if first[0] != '\n' && first[0] != '\r' && first[0] != ' ' && first[0] != '\t' {
			break
		}
		_, _ = reader.ReadByte()
	}

	headerOrJSON, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(headerOrJSON) != "" {
			return []byte(strings.TrimSpace(headerOrJSON)), nil
		}
		return nil, err
	}
	if strings.HasPrefix(strings.ToLower(headerOrJSON), "content-length:") {
		_, lengthValue, _ := strings.Cut(headerOrJSON, ":")
		lengthValue = strings.TrimSpace(lengthValue)
		length, err := strconv.Atoi(lengthValue)
		if err != nil {
			return nil, fmt.Errorf("invalid Content-Length header: %w", err)
		}
		if length < 0 {
			return nil, fmt.Errorf("invalid Content-Length header: must be non-negative")
		}
		if length > maxMCPMessageSize {
			return nil, fmt.Errorf("invalid Content-Length header: exceeds maximum size %d", maxMCPMessageSize)
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(line) == "" {
				break
			}
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		return payload, nil
	}
	return []byte(strings.TrimSpace(headerOrJSON)), nil
}

func (s *stdioServer) writeResponse(resp rpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = s.out.Write(append(data, '\n'))
	return err
}
