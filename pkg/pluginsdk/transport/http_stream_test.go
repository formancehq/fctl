package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const httpStreamTestURL = "https://stream.example/api/ledger"

func newHTTPStreamCallback(t *testing.T, host httpRPC, execution context.Context) *callbackTransport {
	t.Helper()
	listener := bufconn.Listen(128 << 10)
	server := grpc.NewServer(grpc.MaxRecvMsgSize(httpStreamWireBytes), grpc.MaxSendMsgSize(httpStreamWireBytes))
	descriptor := httpDescriptor
	descriptor.Streams = []grpc.StreamDesc{httpStreamDescriptor()}
	server.RegisterService(&descriptor, host)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///http-stream", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDisableRetry(), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}))
	if err != nil {
		server.Stop()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
		server.Stop()
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
		if err := <-served; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Error(err)
		}
	})
	return &callbackTransport{conn: conn, execution: execution}
}

func httpStreamTestHost(t *testing.T, client *http.Client, endpoint string) *hostHTTP {
	t.Helper()
	return &hostHTTP{client: client, endpoint: mustURL(t, endpoint)}
}

func httpStreamTestRequest(t *testing.T, ctx context.Context, method, target string, body io.Reader) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func waitHTTPStreamSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP stream operation did not finish")
	}
}

// The source is generated one Read at a time. After the prefix, a gate prevents
// the producer from finishing before the consumer has seen incremental data.
type httpStreamLazyBody struct {
	prefix    []byte
	remaining int64
	gate      <-chan struct{}
	closed    chan struct{}
	once      sync.Once
	readBytes atomic.Int64
	maxRead   atomic.Int64
}

func newHTTPStreamLazyBody(prefix string, remaining int64, gate <-chan struct{}) *httpStreamLazyBody {
	return &httpStreamLazyBody{prefix: []byte(prefix), remaining: remaining, gate: gate, closed: make(chan struct{})}
}

func (b *httpStreamLazyBody) Read(buffer []byte) (int, error) {
	b.maxRead.Store(max(b.maxRead.Load(), int64(len(buffer))))
	if len(b.prefix) > 0 {
		n := copy(buffer, b.prefix)
		b.prefix = b.prefix[n:]
		b.readBytes.Add(int64(n))
		return n, nil
	}
	if b.remaining == 0 {
		return 0, io.EOF
	}
	select {
	case <-b.gate:
	case <-b.closed:
		return 0, io.ErrClosedPipe
	}
	n := int(min(int64(len(buffer)), b.remaining))
	for i := range n {
		buffer[i] = 'x'
	}
	b.remaining -= int64(n)
	b.readBytes.Add(int64(n))
	return n, nil
}

func (b *httpStreamLazyBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

func TestHTTPStreamLargeLazyResponse(t *testing.T) {
	t.Parallel()
	const prefix = "{\"id\":90071992547409930001}\n"
	const size = int64(40 << 20)
	gate := make(chan struct{})
	body := newHTTPStreamLazyBody(prefix, size, gate)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, ContentLength: size + int64(len(prefix)), Header: http.Header{
			"Content-Type": {"application/x-ndjson"}, "Content-Length": {strconv.FormatInt(size+int64(len(prefix)), 10)},
			"Authorization": {"secret"}, "Set-Cookie": {"secret"}, "X-Request-Id": {"visible"},
		}}, nil
	})}
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
	response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "GET", httpStreamTestURL+"/dump", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.ContentLength != size+int64(len(prefix)) || response.Header.Get("Content-Length") != strconv.FormatInt(response.ContentLength, 10) ||
		response.Header.Get("Authorization") != "" || response.Header.Get("Set-Cookie") != "" || response.Header.Get("X-Request-Id") != "visible" {
		t.Fatalf("response metadata changed: %+v", response)
	}
	first := make([]byte, len(prefix))
	if _, err := io.ReadFull(response.Body, first); err != nil || string(first) != prefix {
		t.Fatalf("incremental prefix = %q, %v", first, err)
	}
	if body.readBytes.Load() != int64(len(prefix)) {
		t.Fatal("response was buffered before incremental delivery")
	}
	close(gate)
	written, err := io.Copy(io.Discard, response.Body)
	if err != nil || written != size || body.maxRead.Load() > httpStreamChunkBytes {
		t.Fatalf("large stream = %d, %v, read size = %d", written, err, body.maxRead.Load())
	}
	waitHTTPStreamSignal(t, body.closed)
}

func TestHTTPStreamRequestAndResponseAreConcurrent(t *testing.T) {
	t.Parallel()
	for _, known := range []bool{true, false} {
		t.Run(strconv.FormatBool(known), func(t *testing.T) {
			t.Parallel()
			testHTTPStreamDuplex(t, known)
		})
	}
}

func testHTTPStreamDuplex(t *testing.T, known bool) {
	t.Helper()
	const prefix = "request-prefix\n"
	const size = int64(40 << 20)
	gate := make(chan struct{})
	input := newHTTPStreamLazyBody(prefix, size, gate)
	finished := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(finished)
		wantLength := int64(-1)
		if known {
			wantLength = size + int64(len(prefix))
		}
		if r.ContentLength != wantLength || r.Header.Get("Content-Type") != "application/x-ndjson" {
			t.Errorf("request metadata = %d, %v", r.ContentLength, r.Header)
		}
		controller := http.NewResponseController(w)
		if err := controller.EnableFullDuplex(); err != nil {
			t.Error(err)
			return
		}
		first := make([]byte, len(prefix))
		if _, err := io.ReadFull(r.Body, first); err != nil || string(first) != prefix {
			t.Errorf("request prefix = %q, %v", first, err)
			return
		}
		if _, err := io.WriteString(w, "accepted\n"); err != nil {
			t.Error(err)
			return
		}
		if err := controller.Flush(); err != nil {
			t.Error(err)
			return
		}
		if n, err := io.Copy(io.Discard, r.Body); err != nil || n != size {
			t.Errorf("request remainder = %d, %v", n, err)
		}
		if _, err := io.WriteString(w, "done\n"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, server.Client(), server.URL), t.Context())
	request := httpStreamTestRequest(t, t.Context(), "POST", server.URL+"/bulk", input)
	request.ContentLength = -1
	if known {
		request.ContentLength = size + int64(len(prefix))
	}
	request.Header.Set("Content-Type", "application/x-ndjson")
	response, err := callback.roundTripStream(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	first := make([]byte, len("accepted\n"))
	if _, err := io.ReadFull(response.Body, first); err != nil || string(first) != "accepted\n" {
		t.Fatalf("response before upload completion = %q, %v", first, err)
	}
	if input.readBytes.Load() != int64(len(prefix)) || response.ContentLength != -1 {
		t.Fatal("upload finished before incremental response or unknown length was lost")
	}
	close(gate)
	last, err := io.ReadAll(response.Body)
	if err != nil || string(last) != "done\n" || input.maxRead.Load() > httpStreamChunkBytes {
		t.Fatalf("duplex completion = %q, %v, read size = %d", last, err, input.maxRead.Load())
	}
	waitHTTPStreamSignal(t, finished)
	waitHTTPStreamSignal(t, input.closed)
}

type httpStreamBlockingBody struct {
	started  chan struct{}
	closed   chan struct{}
	returned chan struct{}
	start    sync.Once
	close    sync.Once
	read     sync.Once
}

func newHTTPStreamBlockingBody() *httpStreamBlockingBody {
	return &httpStreamBlockingBody{started: make(chan struct{}), closed: make(chan struct{}), returned: make(chan struct{})}
}

func (b *httpStreamBlockingBody) Read([]byte) (int, error) {
	b.start.Do(func() { close(b.started) })
	<-b.closed
	b.read.Do(func() { close(b.returned) })
	return 0, io.ErrClosedPipe
}

func (b *httpStreamBlockingBody) Close() error {
	b.close.Do(func() { close(b.closed) })
	return nil
}

func TestHTTPStreamCancellationAndCloseUnblockBodies(t *testing.T) {
	t.Parallel()
	for _, cause := range []string{"request", "execution", "close"} {
		t.Run(cause, func(t *testing.T) {
			t.Parallel()
			requestCtx, cancelRequest := context.WithCancel(t.Context())
			defer cancelRequest()
			executionCtx, cancelExecution := context.WithCancel(t.Context())
			defer cancelExecution()
			input, output := newHTTPStreamBlockingBody(), newHTTPStreamBlockingBody()
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: output, ContentLength: -1}, nil
			})}
			callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), executionCtx)
			response, err := callback.roundTripStream(httpStreamTestRequest(t, requestCtx, "POST", httpStreamTestURL, input))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Error(err)
				}
			}()
			read := make(chan error, 1)
			go func() { _, err := io.Copy(io.Discard, response.Body); read <- err }()
			waitHTTPStreamSignal(t, input.started)
			waitHTTPStreamSignal(t, output.started)
			if cause == "request" {
				cancelRequest()
			} else if cause == "execution" {
				cancelExecution()
			} else if err := response.Body.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-read:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("body cancellation = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("response read did not stop")
			}
			waitHTTPStreamSignal(t, input.returned)
			waitHTTPStreamSignal(t, output.returned)
		})
	}
}

func TestHTTPStreamEarlyResponseStopsUpload(t *testing.T) {
	t.Parallel()
	input := newHTTPStreamBlockingBody()
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusRequestEntityTooLarge, Body: io.NopCloser(strings.NewReader("rejected")), ContentLength: 8}, nil
	})}
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
	response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "POST", httpStreamTestURL, input))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	data, err := io.ReadAll(response.Body)
	if err != nil || string(data) != "rejected" || response.StatusCode != 413 {
		t.Fatalf("early response = %q, %v", data, err)
	}
	waitHTTPStreamSignal(t, input.returned)
}

type httpStreamFailureBody struct {
	readErr  error
	closeErr error
	closed   atomic.Bool
}

func (b *httpStreamFailureBody) Read([]byte) (int, error) { return 0, b.readErr }
func (b *httpStreamFailureBody) Close() error {
	b.closed.Store(true)
	return b.closeErr
}

func TestHTTPStreamFailuresAreRedactedAndNotRetried(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"transport", "read", "close"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			secret := errors.New("Bearer secret /private/host/credentials")
			body := &httpStreamFailureBody{readErr: io.EOF}
			if failure == "read" {
				body.readErr = secret
			} else if failure == "close" {
				body.closeErr = secret
			}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if failure == "transport" {
					return nil, secret
				}
				return &http.Response{StatusCode: 200, Body: body, ContentLength: -1}, nil
			})}
			callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
			response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "GET", httpStreamTestURL, nil))
			if err == nil {
				_, err = io.Copy(io.Discard, response.Body)
				if closeErr := response.Body.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			}
			if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "/private") || calls.Load() != 1 {
				t.Fatalf("failure = %v, attempts = %d", err, calls.Load())
			}
			if failure != "transport" && !body.closed.Load() {
				t.Fatal("failed response body was not closed")
			}
		})
	}
}

func TestHTTPStreamSourceReadErrorIsPreserved(t *testing.T) {
	t.Parallel()
	original := errors.New("source failed")
	input := &httpStreamFailureBody{readErr: original}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		_, err := io.Copy(io.Discard, request.Body)
		return nil, err
	})}
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
	_, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "POST", httpStreamTestURL, input))
	if !errors.Is(err, original) || !input.closed.Load() {
		t.Fatalf("source error = %v, closed = %v", err, input.closed.Load())
	}
}

func TestHTTPStreamEndpointHeadersAndRedirects(t *testing.T) {
	t.Parallel()
	var escaped atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { escaped.Add(1) }))
	t.Cleanup(outside.Close)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ledger/inside":
			http.Redirect(w, r, "/api/ledger/ok", http.StatusFound)
		case "/api/ledger/origin":
			http.Redirect(w, r, outside.URL, http.StatusFound)
		case "/api/ledger/sibling":
			http.Redirect(w, r, "/api/auth/secret", http.StatusFound)
		case "/api/ledger/ok":
			if _, err := io.WriteString(w, "ok"); err != nil {
				t.Error(err)
			}
		default:
			escaped.Add(1)
		}
	}))
	t.Cleanup(server.Close)
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, server.Client(), server.URL+"/api/ledger"), t.Context())
	for _, path := range []string{"/api/auth/secret", "/api/ledger-other", "/api/ledger/../auth", "/api/ledger/%252e%252e/auth", "/api/ledger/%5c..%5cauth", "/api/ledger/origin", "/api/ledger/sibling"} {
		request := httpStreamTestRequest(t, t.Context(), "GET", server.URL+path, nil)
		if response, err := callback.roundTripStream(request); err == nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
			t.Fatalf("endpoint escape accepted: %s", path)
		}
	}
	for _, name := range []string{"Authorization", "Cookie", "Proxy-Authorization", "X-Api-Key", "Host"} {
		request := httpStreamTestRequest(t, t.Context(), "GET", server.URL+"/api/ledger/ok", nil)
		request.Header.Set(name, "secret")
		if response, err := callback.roundTripStream(request); err == nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
			t.Fatalf("credential header accepted: %s", name)
		}
	}
	response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "GET", server.URL+"/api/ledger/inside", nil))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Error(closeErr)
	}
	if err != nil || string(data) != "ok" || escaped.Load() != 0 {
		t.Fatalf("redirect = %q, %v, escapes = %d", data, err, escaped.Load())
	}
}

type httpStreamTestPeer struct {
	*hostHTTP
	exchange func(grpc.ServerStream) error
}

func (p *httpStreamTestPeer) Exchange(stream grpc.ServerStream) error { return p.exchange(stream) }

func TestHTTPStreamRejectsTruncatedAndInvalidResponseFrames(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"missing EOF", "oversized chunk", "empty chunk", "after EOF", "missing header"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			peer := &httpStreamTestPeer{hostHTTP: &hostHTTP{}, exchange: func(stream grpc.ServerStream) error {
				var initial wrapperspb.BytesValue
				if err := stream.RecvMsg(&initial); err != nil {
					return err
				}
				if failure == "missing header" {
					return stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamData, 'x'}))
				}
				header, err := httpStreamMetadata(httpStreamResponse, httpStreamResponseWire{StatusCode: 200, ContentLength: -1})
				if err != nil {
					return err
				}
				if err := stream.SendMsg(header); err != nil {
					return err
				}
				return sendInvalidHTTPStreamFrame(stream, failure)
			}}
			callback := newHTTPStreamCallback(t, peer, t.Context())
			response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "GET", httpStreamTestURL, nil))
			if err == nil {
				_, err = io.Copy(io.Discard, response.Body)
				if closeErr := response.Body.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			}
			if err == nil {
				t.Fatal("invalid response stream accepted")
			}
		})
	}
}

func sendInvalidHTTPStreamFrame(stream grpc.ServerStream, failure string) error {
	switch failure {
	case "missing EOF":
		return nil
	case "oversized chunk":
		return stream.SendMsg(wrapperspb.Bytes(append([]byte{httpStreamData}, make([]byte, httpStreamChunkBytes+1)...)))
	case "empty chunk":
		return stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamData}))
	case "after EOF":
		if err := stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamEOF})); err != nil {
			return err
		}
		return stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamData, 'x'}))
	default:
		return errors.New("invalid test case")
	}
}

// SendMsg fails while the input worker is blocked on its pipe Write. synctest
// fails the test if that worker remains blocked after Exchange returns.
type httpStreamSendFailure struct {
	grpc.ServerStream
	ctx     context.Context
	initial *wrapperspb.BytesValue
	started chan struct{}
}

func (s *httpStreamSendFailure) Context() context.Context { return s.ctx }
func (s *httpStreamSendFailure) RecvMsg(value any) error {
	message, ok := value.(*wrapperspb.BytesValue)
	if !ok {
		return errors.New("invalid test message")
	}
	if s.initial != nil {
		message.Value = s.initial.Value
		s.initial = nil
		return nil
	}
	message.Value = []byte{httpStreamData, 'x'}
	close(s.started)
	return nil
}

func (s *httpStreamSendFailure) SendMsg(any) error {
	<-s.started
	synctest.Wait()
	return status.Error(codes.Unavailable, "peer stopped receiving")
}

func TestHTTPStreamFailedSendUnblocksInputWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		body := &httpStreamFailureBody{readErr: io.EOF}
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: body, ContentLength: -1}, nil
		})}
		initial, err := httpStreamMetadata(httpStreamRequest, httpStreamRequestWire{Method: "POST", URL: httpStreamTestURL, HasBody: true, ContentLength: -1})
		if err != nil {
			t.Fatal(err)
		}
		stream := &httpStreamSendFailure{ctx: t.Context(), initial: initial, started: make(chan struct{})}
		err = httpStreamTestHost(t, client, httpStreamTestURL).Exchange(stream)
		if status.Code(err) != codes.Unavailable || !body.closed.Load() {
			t.Fatalf("failed writer = %v, body closed = %v", err, body.closed.Load())
		}
	})
}

func TestHTTPStreamMetadataLimits(t *testing.T) {
	t.Parallel()
	if _, err := httpStreamMetadata(httpStreamRequest, strings.Repeat("x", maxMetaBytes)); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("oversized metadata = %v", err)
	}
	body := &httpStreamFailureBody{readErr: io.EOF}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: http.Header{"X-Large": {strings.Repeat("x", maxHeadBytes)}}, ContentLength: -1}, nil
	})}
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
	_, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "GET", httpStreamTestURL, nil))
	if status.Code(err) != codes.ResourceExhausted || !body.closed.Load() {
		t.Fatalf("oversized response headers = %v, closed = %v", err, body.closed.Load())
	}
	request := httpStreamTestRequest(t, t.Context(), "GET", httpStreamTestURL, nil)
	request.Header.Set("X-Large", strings.Repeat("x", maxHeadBytes))
	if _, err := callback.roundTripStream(request); err == nil {
		t.Fatal("oversized request headers accepted")
	}
	if err := httpExchangeHandler(struct{}{}, nil); status.Code(err) != codes.Internal {
		t.Fatalf("invalid service implementation = %v", err)
	}
}

func TestHTTPStreamHostPreservesRedirectPolicyAndCannotReplayInput(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			if request.GetBody != nil || request.ContentLength != 3 {
				return nil, fmt.Errorf("stream body can replay or length changed")
			}
			if _, err := io.Copy(io.Discard, request.Body); err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {httpStreamTestURL + "/next"}},
				Body: io.NopCloser(bytes.NewReader(nil)), ContentLength: 0}, nil
		})}
	callback := newHTTPStreamCallback(t, httpStreamTestHost(t, client, httpStreamTestURL), t.Context())
	response, err := callback.roundTripStream(httpStreamTestRequest(t, t.Context(), "POST", httpStreamTestURL, strings.NewReader("abc")))
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusTemporaryRedirect || calls.Load() != 1 {
		t.Fatalf("non-replayable redirect = %d, calls = %d", response.StatusCode, calls.Load())
	}
}
