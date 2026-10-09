package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type httpRequestWire struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header,omitzero"`
	Body   []byte      `json:"body,omitzero"`
}

type httpResponseWire struct {
	StatusCode int         `json:"statusCode"`
	Header     http.Header `json:"header,omitzero"`
	Body       []byte      `json:"body,omitzero"`
	Error      *errorWire  `json:"error,omitzero"`
}

type callbackTransport struct {
	conn      *grpc.ClientConn
	execution context.Context
}

func (c *callbackTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(request.Context())
	//nolint:contextcheck // Merge the request context with its parent Execute RPC.
	stop := context.AfterFunc(c.execution, cancel)
	defer cancel()
	defer stop()
	var body []byte
	if request.Body != nil {
		var err error
		body, err = readLimited(request.Body, maxBodyBytes)
		err = errors.Join(err, request.Body.Close())
		if err != nil {
			return nil, err
		}
	}
	var response httpResponseWire
	if err := invoke(ctx, c.conn, httpService+"/Do", httpRequestWire{request.Method, request.URL.String(), request.Header, body}, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, response.Error.err()
	}
	return &http.Response{StatusCode: response.StatusCode, Status: fmt.Sprintf("%d %s", response.StatusCode, http.StatusText(response.StatusCode)),
		Header: response.Header, Body: io.NopCloser(bytes.NewReader(response.Body)),
		ContentLength: int64(len(response.Body)), Request: request, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}, nil
}

type callbackServer struct {
	id      uint32
	mu      sync.Mutex
	server  *grpc.Server
	stopped bool
}

func (c *Client) serveHTTP() *callbackServer {
	callback := &callbackServer{id: c.remote.broker.NextId()}
	stop := context.AfterFunc(c.lifetime, callback.stop)
	go func() {
		defer stop()
		c.remote.broker.AcceptAndServe(callback.id, func(options []grpc.ServerOption) *grpc.Server {
			server := grpc.NewServer(append(options, grpc.MaxRecvMsgSize(maxWireBytes), grpc.MaxSendMsgSize(maxWireBytes))...)
			server.RegisterService(&httpDescriptor, &hostHTTP{client: c.http, endpoint: c.endpoint})
			callback.mu.Lock()
			callback.server = server
			if callback.stopped {
				server.Stop()
			}
			callback.mu.Unlock()
			return server
		})
	}()
	return callback
}

func (c *callbackServer) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopped = true
	if c.server != nil {
		c.server.Stop()
	}
}

type hostHTTP struct {
	client   *http.Client
	endpoint *url.URL
}

func (h *hostHTTP) Do(ctx context.Context, message *wrapperspb.BytesValue) (result *wrapperspb.BytesValue, err error) {
	var wire httpRequestWire
	if err := unpack(message, &wire); err != nil {
		return nil, err
	}
	request, err := h.buildRequest(ctx, wire)
	if err != nil {
		return nil, err
	}
	// Preserve the injected transport, timeout and cookie jar. CheckRedirect is
	// wrapped so that credentials cannot follow a redirect out of this boundary.
	client := *h.client
	client.CheckRedirect = h.checkRedirect
	response, err := client.Do(request)
	if err != nil {
		// Transport errors can include URLs, credentials or host file paths.
		// Keep cancellation semantics, but never disclose those details.
		failure := &errorWire{Kind: "host", Message: "host HTTP request failed"}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			failure = encodeError(err)
		}
		return pack(httpResponseWire{Error: failure})
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			err = errors.Join(err, status.Error(codes.Unavailable, "host HTTP response close failed"))
		}
	}()
	payload, err := readResponse(response)
	if err != nil {
		return nil, err
	}
	return pack(payload)
}

func (h *hostHTTP) buildRequest(ctx context.Context, wire httpRequestWire) (*http.Request, error) {
	if len(wire.Body) > maxBodyBytes || headerSize(wire.Header) > maxHeadBytes {
		return nil, status.Error(codes.ResourceExhausted, "plugin HTTP request exceeds transport limit")
	}
	target, err := url.Parse(wire.URL)
	if err != nil || validateEndpoint(h.endpoint, target) != nil {
		return nil, status.Error(codes.PermissionDenied, "plugin HTTP request is outside its endpoint")
	}
	for name := range wire.Header {
		if credentialKey(name) || strings.EqualFold(name, "Host") {
			return nil, status.Error(codes.PermissionDenied, "plugin HTTP headers must not contain credentials or override Host")
		}
	}
	request, err := http.NewRequestWithContext(ctx, wire.Method, target.String(), bytes.NewReader(wire.Body))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plugin HTTP request")
	}
	request.Header = wire.Header.Clone()
	return request, nil
}

func (h *hostHTTP) checkRedirect(next *http.Request, previous []*http.Request) error {
	if err := validateEndpoint(h.endpoint, next.URL); err != nil {
		return err
	}
	if h.client.CheckRedirect != nil {
		return h.client.CheckRedirect(next, previous)
	}
	if len(previous) >= 10 {
		return fmt.Errorf("too many service redirects")
	}
	return nil
}

func readResponse(response *http.Response) (payload httpResponseWire, err error) {
	body, err := readLimited(response.Body, maxDataBytes)
	if err != nil {
		return payload, status.Error(codes.ResourceExhausted, "host HTTP response exceeds transport limit or cannot be read")
	}
	headers := response.Header.Clone()
	for name := range headers {
		if credentialKey(name) {
			delete(headers, name)
		}
	}
	if headerSize(headers) > maxHeadBytes {
		return payload, status.Error(codes.ResourceExhausted, "host HTTP response headers exceed transport limit")
	}
	return httpResponseWire{StatusCode: response.StatusCode, Header: headers, Body: body}, nil
}

func readLimited(reader io.Reader, limit int) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("HTTP body exceeds transport limit")
	}
	return body, nil
}

func headerSize(headers http.Header) int {
	size := 0
	for name, values := range headers {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}
	return size
}

func credentialKey(name string) bool {
	name = strings.ToLower(name)
	name = strings.NewReplacer("-", "", "_", "", " ", "").Replace(name)
	switch name {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "token", "accesstoken", "refreshtoken", "idtoken", "clientsecret", "password", "apikey", "xapikey", "xauthtoken":
		return true
	default:
		return false
	}
}

func validateEndpoint(base, target *url.URL) error {
	if target == nil || !strings.EqualFold(target.Scheme, base.Scheme) || !strings.EqualFold(target.Host, base.Host) || target.User != nil || target.Fragment != "" || target.Opaque != "" {
		return fmt.Errorf("plugin HTTP request is outside its endpoint")
	}
	if err := validatePath(target); err != nil {
		return err
	}
	basePath := strings.TrimRight(base.Path, "/")
	if target.Path != basePath && !strings.HasPrefix(target.Path, basePath+"/") {
		return fmt.Errorf("plugin HTTP request is outside its endpoint path")
	}
	return nil
}

func validatePath(target *url.URL) error {
	// Reject traversal even through encoded separators or double decoding in
	// a gateway. Encoded slashes in ordinary service IDs remain supported.
	value := target.EscapedPath()
	for {
		if strings.Contains(value, "\\") {
			return fmt.Errorf("plugin endpoint paths must not contain backslashes")
		}
		for segment := range strings.SplitSeq(value, "/") {
			if segment == "." || segment == ".." {
				return fmt.Errorf("plugin endpoint paths must not contain dot segments")
			}
		}
		decoded, err := url.PathUnescape(value)
		if err != nil || decoded == value {
			return nil
		}
		value = decoded
	}
}
