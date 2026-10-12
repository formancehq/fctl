package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

const streamChunkBytes = 64 << 10

type streamStart struct {
	Execution executeWire `json:"execution"`
	Input     bool        `json:"input,omitzero"`
	InputSize *int64      `json:"inputSize,omitzero"`
	Output    bool        `json:"output,omitzero"`
}

// Input is pulled by the plugin: one bounded chunk per request. There is no
// background uploader, unbounded queue, or read-ahead before business checks.
type streamFrame struct {
	Start    *streamStart             `json:"start,omitzero"`
	Ready    bool                     `json:"ready,omitzero"`
	Pull     bool                     `json:"pull,omitzero"`
	Data     []byte                   `json:"data,omitzero"`
	End      bool                     `json:"end,omitzero"`
	Failure  *errorWire               `json:"failure,omitzero"`
	Progress *pluginsdk.ProgressEvent `json:"progress,omitzero"`
	Result   *resultWire              `json:"result,omitzero"`
}

type frameSender interface{ SendMsg(any) error }
type frameReceiver interface{ RecvMsg(any) error }

func sendFrame(stream frameSender, frame streamFrame) error {
	message, err := pack(frame)
	if err != nil {
		return err
	}
	return stream.SendMsg(message)
}

func receiveFrame(stream frameReceiver) (streamFrame, error) {
	var message wrapperspb.BytesValue
	if err := stream.RecvMsg(&message); err != nil {
		return streamFrame{}, err
	}
	var frame streamFrame
	if err := unpack(&message, &frame); err != nil {
		return streamFrame{}, err
	}
	if len(frame.Data) > streamChunkBytes {
		return streamFrame{}, status.Error(codes.ResourceExhausted, "plugin stream chunk exceeds 64 KiB")
	}
	kinds := 0
	for _, present := range []bool{frame.Start != nil, frame.Ready, frame.Pull, frame.Data != nil,
		frame.End, frame.Failure != nil, frame.Progress != nil, frame.Result != nil} {
		if present {
			kinds++
		}
	}
	if kinds != 1 {
		return streamFrame{}, status.Error(codes.InvalidArgument, "invalid plugin stream frame")
	}
	return frame, nil
}

var executeStreamDescriptor = grpc.StreamDesc{
	StreamName: "ExecuteStream", ServerStreams: true, ClientStreams: true,
	Handler: func(server any, stream grpc.ServerStream) error {
		implementation, ok := server.(*pluginServer)
		if !ok {
			return status.Error(codes.Internal, "invalid plugin stream implementation")
		}
		return implementation.ExecuteStream(stream)
	},
}

// ExecuteStream keeps the final bounded JSON result separate from raw output
// chunks and progress. It does not retry or replay reads, writes or operations.
func (c *Client) ExecuteStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (response pluginsdk.ExecuteResponse, err error) {
	request, err = c.streamRequest(ctx, request, streams)
	if err != nil {
		return response, err
	}
	stop := context.AfterFunc(ctx, c.close)
	defer stop()
	callCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// A caller-owned interruptible reader is closed only on cancellation. A
	// successful call leaves it to its owner, just like an ordinary io.Reader.
	if input, ok := streams.Input.(io.ReadCloser); ok {
		interrupt := context.AfterFunc(callCtx, func() {
			//nolint:errcheck // Cancellation already propagates via the RPC; Close only interrupts a blocked Read.
			_ = input.Close()
		})
		defer interrupt()
	}
	callback := c.serveHTTP()
	defer callback.stop()
	stream, err := c.remote.conn.NewStream(callCtx, &executeStreamDescriptor, "/"+pluginService+"/ExecuteStream",
		grpc.MaxCallRecvMsgSize(maxWireBytes), grpc.MaxCallSendMsgSize(maxWireBytes))
	if err != nil {
		return response, c.rpcError(ctx, err)
	}
	defer func() { err = errors.Join(err, stream.CloseSend()) }()
	start := &streamStart{Execution: executeWire{BrokerID: callback.id, Request: toWire(request)},
		Input: streams.Input != nil, InputSize: streams.InputSize, Output: streams.Output != nil}
	if err := sendFrame(stream, streamFrame{Start: start}); err != nil {
		return response, c.rpcError(ctx, err)
	}
	ready, err := receiveFrame(stream)
	if err != nil {
		return response, c.rpcError(ctx, err)
	}
	if !ready.Ready {
		return response, fmt.Errorf("plugin did not acknowledge stream execution")
	}
	result, err := consumeExecution(stream, streams)
	if err != nil {
		return pluginsdk.ExecuteResponse{Data: result.Data}, errors.Join(result.Error.err(), c.rpcError(ctx, err))
	}
	return pluginsdk.ExecuteResponse{Data: result.Data}, result.Error.err()
}

func (c *Client) streamRequest(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteRequest, error) {
	if err := c.check(ctx); err != nil {
		return request, err
	}
	if c.http == nil || c.endpoint == nil {
		return request, fmt.Errorf("plugin execution requires a host HTTP client and endpoint")
	}
	if request.Endpoint == "" {
		request.Endpoint = c.endpoint.String()
	}
	if strings.TrimRight(request.Endpoint, "/") != c.endpoint.String() {
		return request, fmt.Errorf("plugin endpoint differs from host endpoint")
	}
	if err := validateRequest(request); err != nil {
		return request, err
	}
	if streams.Input != nil && request.Body != nil {
		return request, fmt.Errorf("plugin stream input cannot be combined with Body")
	}
	if streams.InputSize != nil && (streams.Input == nil || *streams.InputSize < 0) {
		return request, fmt.Errorf("invalid plugin input size")
	}
	return request, nil
}

func consumeExecution(stream grpc.ClientStream, streams pluginsdk.ExecutionStreams) (resultWire, error) {
	input := &inputSender{reader: streams.Input}
	for {
		frame, err := receiveFrame(stream)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return resultWire{}, err
		}
		if err := consumeFrame(stream, frame, streams, input); err != nil {
			return resultWire{}, err
		}
		if frame.Result != nil {
			if input.err != nil && !errors.Is(input.err, io.EOF) {
				return *frame.Result, input.err
			}
			return *frame.Result, nil
		}
	}
}

func consumeFrame(stream grpc.ClientStream, frame streamFrame, streams pluginsdk.ExecutionStreams, input *inputSender) error {
	switch {
	case frame.Pull:
		return input.send(stream)
	case frame.Data != nil:
		if streams.Output == nil {
			return fmt.Errorf("plugin wrote an undeclared output stream")
		}
		written, err := streams.Output.Write(frame.Data)
		if err == nil && written != len(frame.Data) {
			return io.ErrShortWrite
		}
		return err
	case frame.Progress != nil:
		if err := frame.Progress.Validate(); err != nil {
			return err
		}
		if streams.Report != nil {
			return streams.Report(*frame.Progress)
		}
		return nil
	case frame.Result != nil:
		return validateResult(*frame.Result)
	default:
		return fmt.Errorf("unexpected plugin execution frame")
	}
}

type inputSender struct {
	reader io.Reader
	err    error
	buffer [streamChunkBytes]byte
}

func (i *inputSender) send(stream frameSender) error {
	if i.reader == nil {
		return fmt.Errorf("plugin requested an undeclared input stream")
	}
	if i.err != nil {
		if errors.Is(i.err, io.EOF) {
			return sendFrame(stream, streamFrame{End: true})
		}
		failure := &errorWire{Kind: "input", Message: "host input read failed"}
		if errors.Is(i.err, context.Canceled) || errors.Is(i.err, context.DeadlineExceeded) {
			failure = encodeError(i.err)
		}
		return sendFrame(stream, streamFrame{Failure: failure})
	}
	for range 100 {
		n, err := i.reader.Read(i.buffer[:])
		i.err = err
		if n > 0 {
			return sendFrame(stream, streamFrame{Data: i.buffer[:n]})
		}
		if err != nil {
			return i.send(stream)
		}
	}
	i.err = io.ErrNoProgress
	return i.send(stream)
}

func (s *pluginServer) ExecuteStream(stream grpc.ServerStream) (err error) {
	start, request, err := s.streamRequest(stream)
	if err != nil {
		return err
	}
	conn, err := s.broker.DialWithOptions(start.Execution.BrokerID, grpc.WithDisableRetry())
	if err != nil {
		return status.Error(codes.Unavailable, "connect host HTTP callback")
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	client := &http.Client{Transport: &callbackTransport{conn: conn, execution: stream.Context(), streaming: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	plugin, ok := s.factory(client).(pluginsdk.StreamingPlugin)
	if !ok {
		return status.Error(codes.Unimplemented, "plugin does not implement streaming execution")
	}
	sender := &executionSender{stream: stream}
	if err := sender.send(streamFrame{Ready: true}); err != nil {
		return err
	}
	streams := pluginsdk.ExecutionStreams{InputSize: start.InputSize, Report: sender.report}
	if start.Input {
		streams.Input = &executionReader{stream: stream, sender: sender}
	}
	if start.Output {
		streams.Output = sender
	}
	result, executeErr := plugin.ExecuteStream(stream.Context(), request, streams)
	final := resultWire{Data: result.Data, Error: encodeError(executeErr)}
	if err := validateResult(final); err != nil {
		return status.Error(codes.ResourceExhausted, "invalid final plugin stream result")
	}
	return sender.send(streamFrame{Result: &final})
}

func (s *pluginServer) streamRequest(stream grpc.ServerStream) (*streamStart, pluginsdk.ExecuteRequest, error) {
	frame, err := receiveFrame(stream)
	if err != nil {
		return nil, pluginsdk.ExecuteRequest{}, err
	}
	if frame.Start == nil || frame.Start.Execution.BrokerID == 0 {
		return nil, pluginsdk.ExecuteRequest{}, status.Error(codes.InvalidArgument, "missing plugin stream start or HTTP callback")
	}
	start := frame.Start
	request := start.Execution.Request.request()
	if err := validateRequest(request); err != nil {
		return nil, pluginsdk.ExecuteRequest{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if (start.Input && request.Body != nil) || (start.InputSize != nil && (!start.Input || *start.InputSize < 0)) {
		return nil, pluginsdk.ExecuteRequest{}, status.Error(codes.InvalidArgument, "invalid plugin stream input")
	}
	manifest, err := s.factory(nil).GetManifest(stream.Context())
	if err != nil {
		return nil, pluginsdk.ExecuteRequest{}, err
	}
	request, err = pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return nil, pluginsdk.ExecuteRequest{}, err
	}
	spec, err := pluginsdk.FindCommand(manifest, request.CommandPath)
	if err != nil || spec.Streams == nil || (start.Input && !spec.Streams.Input) || (start.Output && !spec.Streams.Output) {
		return nil, pluginsdk.ExecuteRequest{}, status.Error(codes.InvalidArgument, "plugin command does not declare the requested streams")
	}
	return start, request, nil
}

type executionSender struct {
	stream grpc.ServerStream
	mu     sync.Mutex
}

func (s *executionSender) send(frame streamFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return sendFrame(s.stream, frame)
}

func (s *executionSender) Write(data []byte) (int, error) {
	written := 0
	for len(data) > 0 {
		n := min(len(data), streamChunkBytes)
		if err := s.send(streamFrame{Data: data[:n]}); err != nil {
			return written, err
		}
		written += n
		data = data[n:]
	}
	return written, nil
}

func (s *executionSender) report(progress pluginsdk.ProgressEvent) error {
	if err := progress.Validate(); err != nil {
		return err
	}
	return s.send(streamFrame{Progress: &progress})
}

type executionReader struct {
	stream grpc.ServerStream
	sender *executionSender
	buffer []byte
	end    bool
}

func (r *executionReader) Read(target []byte) (int, error) {
	if len(target) == 0 {
		return 0, nil
	}
	if len(r.buffer) == 0 && !r.end {
		if err := r.sender.send(streamFrame{Pull: true}); err != nil {
			return 0, err
		}
		frame, err := receiveFrame(r.stream)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
		switch {
		case frame.End:
			r.end = true
		case frame.Failure != nil:
			return 0, frame.Failure.err()
		case len(frame.Data) != 0:
			r.buffer = frame.Data
		default:
			return 0, fmt.Errorf("unexpected plugin input frame")
		}
	}
	if r.end {
		return 0, io.EOF
	}
	n := copy(target, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}
