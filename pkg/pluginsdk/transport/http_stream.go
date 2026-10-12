package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	httpStreamChunkBytes = 64 << 10
	httpStreamWireBytes  = maxMetaBytes + 64 // Metadata plus protobuf framing.
	httpStreamRequest    = byte(1)
	httpStreamResponse   = byte(2)
	httpStreamData       = byte(3)
	httpStreamEOF        = byte(4)
)

// Exchange is additive to the v1 unary Do RPC. Each BytesValue starts with a
// frame kind. The first frame in each direction contains JSON metadata; data
// frames contain up to 64 KiB of raw bytes, followed by an explicit EOF frame.
// gRPC completion must also succeed: EOF alone cannot hide a final read/close
// failure. Neither request nor response bodies pass through JSON or pack.
type httpStreamRequestWire struct {
	Method        string      `json:"method"`
	URL           string      `json:"url"`
	Header        http.Header `json:"header,omitzero"`
	ContentLength int64       `json:"contentLength"`
	HasBody       bool        `json:"hasBody"`
}

type httpStreamResponseWire struct {
	StatusCode    int         `json:"statusCode"`
	Header        http.Header `json:"header,omitzero"`
	ContentLength int64       `json:"contentLength"`
	Error         *errorWire  `json:"error,omitzero"`
}

type httpStreamRPC interface {
	Exchange(grpc.ServerStream) error
}

// Append this descriptor to httpDescriptor.Streams when registering the host
// callback. Keeping a separate descriptor leaves the unary service unchanged.
func httpStreamDescriptor() grpc.StreamDesc {
	return grpc.StreamDesc{StreamName: "Exchange", Handler: httpExchangeHandler, ServerStreams: true, ClientStreams: true}
}

func httpExchangeHandler(server any, stream grpc.ServerStream) error {
	implementation, ok := server.(httpStreamRPC)
	if !ok {
		return status.Error(codes.Internal, "invalid streaming HTTP RPC implementation")
	}
	return implementation.Exchange(stream)
}

func httpStreamMetadata(kind byte, value any) (*wrapperspb.BytesValue, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid HTTP stream metadata")
	}
	if len(data) > maxMetaBytes {
		return nil, status.Error(codes.ResourceExhausted, "HTTP stream metadata exceeds transport limit")
	}
	return wrapperspb.Bytes(append([]byte{kind}, data...)), nil
}

func decodeHTTPStreamMetadata(message *wrapperspb.BytesValue, kind byte, value any) error {
	if message == nil || len(message.Value) > maxMetaBytes+1 {
		return status.Error(codes.ResourceExhausted, "HTTP stream metadata exceeds transport limit")
	}
	if len(message.Value) < 2 || message.Value[0] != kind || json.Unmarshal(message.Value[1:], value) != nil {
		return status.Error(codes.InvalidArgument, "invalid HTTP stream metadata")
	}
	return nil
}

type httpFrameReceiver interface {
	RecvMsg(any) error
}

func receiveHTTPStreamFrame(receiver httpFrameReceiver) ([]byte, error) {
	var message wrapperspb.BytesValue
	if err := receiver.RecvMsg(&message); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if len(message.Value) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty HTTP stream frame")
	}
	switch message.Value[0] {
	case httpStreamData:
		if len(message.Value) > httpStreamChunkBytes+1 {
			return nil, status.Error(codes.ResourceExhausted, "HTTP stream chunk exceeds 64 KiB")
		}
		if len(message.Value) == 1 {
			return nil, status.Error(codes.InvalidArgument, "empty HTTP stream chunk")
		}
	case httpStreamEOF:
		if len(message.Value) != 1 {
			return nil, status.Error(codes.InvalidArgument, "invalid HTTP stream EOF")
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unexpected HTTP stream frame")
	}
	return message.Value, nil
}

// roundTripStream keeps its context alive until the response body is consumed
// or closed. A request body must obey net/http's concurrent Read/Close contract,
// so closing it interrupts a blocked Read on cancellation or an early response.
func (c *callbackTransport) roundTripStream(request *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancelCause(request.Context())
	//nolint:contextcheck // Merge the HTTP request with its parent Execute RPC.
	stopExecution := context.AfterFunc(c.execution, func() { cancel(context.Cause(c.execution)) })
	closeInput := sync.OnceFunc(func() {
		if request.Body != nil {
			if err := request.Body.Close(); err != nil {
				cancel(err)
			}
		}
	})
	stopInput := context.AfterFunc(ctx, closeInput)
	finish := sync.OnceFunc(func() {
		cancel(context.Canceled)
		stopExecution()
		stopInput()
		closeInput()
	})
	response, err := c.openHTTPStream(ctx, request, cancel, closeInput, finish)
	if err != nil {
		finish()
	}
	return response, err
}

func (c *callbackTransport) openHTTPStream(ctx context.Context, request *http.Request, cancel context.CancelCauseFunc, closeInput, finish func()) (*http.Response, error) {
	metadata := httpStreamRequestWire{Method: request.Method, URL: request.URL.String(), Header: request.Header,
		ContentLength: request.ContentLength, HasBody: request.Body != nil && request.Body != http.NoBody}
	if metadata.ContentLength < -1 || (!metadata.HasBody && metadata.ContentLength > 0) {
		return nil, status.Error(codes.InvalidArgument, "invalid HTTP stream request metadata")
	}
	if headerSize(metadata.Header) > maxHeadBytes {
		return nil, status.Error(codes.ResourceExhausted, "HTTP stream request headers exceed transport limit")
	}
	message, err := httpStreamMetadata(httpStreamRequest, metadata)
	if err != nil {
		return nil, err
	}
	descriptor := httpStreamDescriptor()
	// Connections are opened with WithDisableRetry. There is one Exchange call,
	// no fallback to unary, no body replay and no application retry loop.
	stream, err := c.conn.NewStream(ctx, &descriptor, "/"+httpService+"/Exchange",
		grpc.MaxCallRecvMsgSize(httpStreamWireBytes), grpc.MaxCallSendMsgSize(httpStreamWireBytes))
	if err != nil {
		return nil, httpStreamError(ctx, err)
	}
	if err := stream.SendMsg(message); err != nil && !errors.Is(err, io.EOF) {
		return nil, httpStreamError(ctx, err)
	}
	go func() {
		defer closeInput()
		sendHTTPStreamBody(stream, request.Body, cancel)
	}()
	var header wrapperspb.BytesValue
	if err := stream.RecvMsg(&header); err != nil {
		return nil, httpStreamError(ctx, err)
	}
	var wire httpStreamResponseWire
	if err := decodeHTTPStreamMetadata(&header, httpStreamResponse, &wire); err != nil {
		return nil, err
	}
	if wire.Error != nil {
		return nil, wire.Error.err()
	}
	if wire.StatusCode < 100 || wire.StatusCode > 999 || wire.ContentLength < -1 || headerSize(wire.Header) > maxHeadBytes {
		return nil, status.Error(codes.InvalidArgument, "invalid HTTP stream response metadata")
	}
	return &http.Response{StatusCode: wire.StatusCode, Status: fmt.Sprintf("%d %s", wire.StatusCode, http.StatusText(wire.StatusCode)),
		Header: wire.Header, ContentLength: wire.ContentLength, Request: request, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
		Body: &httpStreamBody{stream: stream, ctx: ctx, finish: finish}}, nil
}

type httpStreamSender interface {
	SendMsg(any) error
}

type httpStreamCopyResult struct {
	readErr error
	sendErr error
}

func copyHTTPStreamChunks(sender httpStreamSender, body io.Reader) httpStreamCopyResult {
	if body == nil {
		return httpStreamCopyResult{readErr: io.EOF}
	}
	buffer := make([]byte, httpStreamChunkBytes+1)
	buffer[0] = httpStreamData
	for {
		n, err := body.Read(buffer[1:])
		if n > 0 {
			// gRPC may retain a message for tracing after SendMsg returns.
			if sendErr := sender.SendMsg(wrapperspb.Bytes(bytes.Clone(buffer[:n+1]))); sendErr != nil {
				return httpStreamCopyResult{readErr: err, sendErr: sendErr}
			}
		}
		if err != nil {
			return httpStreamCopyResult{readErr: err}
		}
	}
}

// RecvMsg owns send failures and the final status. Only a local source Read
// failure cancels that receive; EOF and early HTTP responses are normal.
func sendHTTPStreamBody(stream grpc.ClientStream, body io.Reader, cancel context.CancelCauseFunc) {
	result := copyHTTPStreamChunks(stream, body)
	if result.readErr != nil && !errors.Is(result.readErr, io.EOF) {
		cancel(result.readErr)
		return
	}
	if result.sendErr != nil {
		return
	}
	if err := stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamEOF})); err != nil {
		return
	}
	// RecvMsg owns the final status even when CloseSend fails.
	if err := stream.CloseSend(); err != nil {
		return
	}
}

type httpStreamBody struct {
	stream   grpc.ClientStream
	ctx      context.Context
	finish   func()
	pending  []byte
	terminal error // Only Read mutates these fields; Close only cancels the RPC.
}

func (b *httpStreamBody) Read(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	if b.terminal != nil {
		return 0, b.terminal
	}
	if b.ctx.Err() != nil {
		return 0, context.Cause(b.ctx)
	}
	if len(b.pending) == 0 {
		frame, err := receiveHTTPStreamFrame(b.stream)
		if err == nil && frame[0] == httpStreamEOF {
			err = b.receiveCompletion()
		}
		if err != nil {
			b.terminal = httpStreamError(b.ctx, err)
			b.finish()
			return 0, b.terminal
		}
		b.pending = frame[1:]
	}
	n := copy(buffer, b.pending)
	b.pending = b.pending[n:]
	return n, nil
}

func (b *httpStreamBody) receiveCompletion() error {
	var extra wrapperspb.BytesValue
	if err := b.stream.RecvMsg(&extra); err != nil {
		return err
	}
	return status.Error(codes.InvalidArgument, "HTTP stream frame after EOF")
}

func (b *httpStreamBody) Close() error {
	b.finish()
	return nil
}

func httpStreamError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if status.Code(err) == codes.Canceled {
		return context.Canceled
	}
	if status.Code(err) == codes.DeadlineExceeded {
		return context.DeadlineExceeded
	}
	return err
}

// Exchange validates only metadata with buildRequest. An unbuffered pipe feeds
// the injected host transport while the handler sends the response concurrently.
// Closing the pipe unblocks Write; returning cancels gRPC's remaining RecvMsg.
func (h *hostHTTP) Exchange(stream grpc.ServerStream) error {
	ctx, cancel := context.WithCancelCause(stream.Context())
	defer cancel(context.Canceled)
	request, wire, err := h.receiveHTTPStreamRequest(ctx, stream)
	if err != nil {
		return err
	}
	closeInput := pipeHTTPStreamInput(ctx, stream, request, wire, cancel)
	defer closeInput()
	client := *h.client
	client.CheckRedirect = h.checkRedirect
	response, err := client.Do(request) //nolint:bodyclose // sendHTTPStreamResponse owns and closes the body on every path.
	if err != nil {
		return sendHTTPStreamFailure(ctx, stream, err)
	}
	return sendHTTPStreamResponse(ctx, stream, response)
}

func (h *hostHTTP) receiveHTTPStreamRequest(ctx context.Context, stream grpc.ServerStream) (*http.Request, httpStreamRequestWire, error) {
	var wire httpStreamRequestWire
	var message wrapperspb.BytesValue
	if err := stream.RecvMsg(&message); err != nil {
		return nil, wire, err
	}
	if err := decodeHTTPStreamMetadata(&message, httpStreamRequest, &wire); err != nil {
		return nil, wire, err
	}
	if wire.ContentLength < -1 || (!wire.HasBody && wire.ContentLength > 0) {
		return nil, wire, status.Error(codes.InvalidArgument, "invalid HTTP stream content length")
	}
	request, err := h.buildRequest(ctx, httpRequestWire{Method: wire.Method, URL: wire.URL, Header: wire.Header})
	if err != nil {
		return nil, wire, err
	}
	request.ContentLength = wire.ContentLength
	request.GetBody = nil // Streaming input is never replayable, including redirects.
	if !wire.HasBody {
		if err := receiveHTTPStreamInput(stream, io.Discard, wire); err != nil {
			return nil, wire, err
		}
	}
	return request, wire, nil
}

func pipeHTTPStreamInput(ctx context.Context, stream grpc.ServerStream, request *http.Request, wire httpStreamRequestWire, cancel context.CancelCauseFunc) func() {
	if !wire.HasBody {
		return func() {}
	}
	reader, writer := io.Pipe()
	closePipe := sync.OnceFunc(func() {
		if err := reader.CloseWithError(context.Canceled); err != nil {
			cancel(err)
		}
		if err := writer.CloseWithError(context.Canceled); err != nil {
			cancel(err)
		}
	})
	stopPipe := context.AfterFunc(ctx, closePipe)
	if wire.HasBody {
		request.Body = reader
	}
	go func() {
		err := receiveHTTPStreamInput(stream, writer, wire)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			cancel(err)
		}
		if closeErr := writer.CloseWithError(err); closeErr != nil {
			cancel(closeErr)
		}
	}()
	return func() {
		stopPipe()
		closePipe()
	}
}

func sendHTTPStreamFailure(ctx context.Context, stream grpc.ServerStream, failure error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	wire := &errorWire{Kind: "host", Message: "host HTTP request failed"}
	if errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) {
		wire = encodeError(failure)
	}
	header, err := httpStreamMetadata(httpStreamResponse, httpStreamResponseWire{Error: wire})
	if err != nil {
		return err
	}
	return stream.SendMsg(header)
}

func receiveHTTPStreamInput(stream grpc.ServerStream, writer io.Writer, wire httpStreamRequestWire) error {
	var received int64
	for {
		frame, err := receiveHTTPStreamFrame(stream)
		if err != nil {
			return err
		}
		if frame[0] == httpStreamEOF {
			if wire.HasBody && wire.ContentLength > 0 && received != wire.ContentLength {
				return status.Error(codes.InvalidArgument, "HTTP stream input length mismatch")
			}
			return completeHTTPStreamInput(stream)
		}
		if !wire.HasBody || (wire.ContentLength > 0 && int64(len(frame)-1) > wire.ContentLength-received) {
			return status.Error(codes.InvalidArgument, "unexpected HTTP stream input data")
		}
		if _, err := writer.Write(frame[1:]); err != nil {
			return err
		}
		received += int64(len(frame) - 1)
	}
}

func completeHTTPStreamInput(stream grpc.ServerStream) error {
	var extra wrapperspb.BytesValue
	err := stream.RecvMsg(&extra)
	if err == nil {
		return status.Error(codes.InvalidArgument, "HTTP stream input frame after EOF")
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func sendHTTPStreamResponse(ctx context.Context, stream grpc.ServerStream, response *http.Response) (err error) {
	closeBody := sync.OnceValue(response.Body.Close)
	stopBody := context.AfterFunc(ctx, func() {
		// The deferred close reports failures; cancellation must interrupt Read.
		if closeErr := closeBody(); closeErr != nil {
			return
		}
	})
	defer stopBody()
	defer func() {
		if closeErr := closeBody(); closeErr != nil {
			err = errors.Join(err, status.Error(codes.Unavailable, "host HTTP response close failed"))
		}
	}()
	headers := response.Header.Clone()
	for name := range headers {
		if credentialKey(name) {
			delete(headers, name)
		}
	}
	if headerSize(headers) > maxHeadBytes {
		return status.Error(codes.ResourceExhausted, "host HTTP response headers exceed transport limit")
	}
	header, err := httpStreamMetadata(httpStreamResponse, httpStreamResponseWire{
		StatusCode: response.StatusCode, Header: headers, ContentLength: response.ContentLength,
	})
	if err != nil {
		return err
	}
	if err := stream.SendMsg(header); err != nil {
		return err
	}
	result := copyHTTPStreamChunks(stream, response.Body)
	if result.sendErr != nil {
		return result.sendErr
	}
	if !errors.Is(result.readErr, io.EOF) {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		return status.Error(codes.Unavailable, "host HTTP response read failed")
	}
	return stream.SendMsg(wrapperspb.Bytes([]byte{httpStreamEOF}))
}
