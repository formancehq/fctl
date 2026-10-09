package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

const (
	pluginService = "formance.fctl.plugins.v1.Plugin"
	httpService   = "formance.fctl.plugins.v1.HTTP"
	maxWireBytes  = 48 << 20 // Includes base64 framing of a 32 MiB response.
	maxBodyBytes  = 4 << 20
	maxDataBytes  = 32 << 20
	maxMetaBytes  = 1 << 20
	maxHeadBytes  = 64 << 10
)

// Byte slices, rather than RawMessage or protobuf Struct, keep service JSON
// byte-for-byte intact, including integers larger than IEEE-754 can represent.
type requestWire struct {
	CommandPath  []string          `json:"commandPath"`
	Args         []string          `json:"args,omitzero"`
	Flags        map[string]string `json:"flags,omitzero"`
	ChangedFlags map[string]bool   `json:"changedFlags,omitzero"`
	Body         []byte            `json:"body,omitzero"`
	Endpoint     string            `json:"endpoint"`
	Context      map[string]string `json:"context,omitzero"`
}

func toWire(r pluginsdk.ExecuteRequest) requestWire {
	return requestWire{r.CommandPath, r.Args, r.Flags, r.ChangedFlags, r.Body, r.Endpoint, r.Context}
}

func (r requestWire) request() pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{CommandPath: r.CommandPath, Args: r.Args, Flags: r.Flags,
		ChangedFlags: r.ChangedFlags, Body: r.Body, Endpoint: r.Endpoint, Context: r.Context}
}

type executeWire struct {
	BrokerID uint32      `json:"brokerID"`
	Request  requestWire `json:"request"`
}

type resultWire struct {
	Data  []byte     `json:"data,omitzero"`
	Error *errorWire `json:"error,omitzero"`
}

type errorWire struct {
	Kind       string `json:"kind"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitzero"`
	Code       string `json:"code,omitzero"`
	Body       []byte `json:"body,omitzero"`
}

func encodeError(err error) *errorWire {
	if err == nil {
		return nil
	}
	if failure, ok := errors.AsType[*httpclient.Error](err); ok {
		return &errorWire{Kind: "http", Message: failure.Message, StatusCode: failure.StatusCode, Code: failure.Code, Body: failure.Body}
	}
	if errors.Is(err, context.Canceled) {
		return &errorWire{Kind: "canceled", Message: context.Canceled.Error()}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &errorWire{Kind: "deadline", Message: context.DeadlineExceeded.Error()}
	}
	return &errorWire{Kind: "plugin", Message: err.Error()}
}

func (e *errorWire) err() error {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case "http":
		return &httpclient.Error{StatusCode: e.StatusCode, Code: e.Code, Message: e.Message, Body: e.Body}
	case "canceled":
		return context.Canceled
	case "deadline":
		return context.DeadlineExceeded
	default:
		return errors.New(e.Message)
	}
}

func pack(value any) (*wrapperspb.BytesValue, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode plugin message: %w", err)
	}
	if len(data) > maxWireBytes {
		return nil, status.Error(codes.ResourceExhausted, "plugin message exceeds transport limit")
	}
	return wrapperspb.Bytes(data), nil
}

func unpack(message *wrapperspb.BytesValue, value any) error {
	if message == nil || len(message.Value) > maxWireBytes {
		return status.Error(codes.ResourceExhausted, "plugin message exceeds transport limit")
	}
	if err := json.Unmarshal(message.Value, value); err != nil {
		return status.Error(codes.InvalidArgument, "invalid plugin transport message")
	}
	return nil
}

func validateResult(result resultWire) error {
	if len(result.Data) > maxDataBytes || (len(result.Data) != 0 && !json.Valid(result.Data)) {
		return fmt.Errorf("plugin response must be valid JSON within 32 MiB")
	}
	size := len(result.Data)
	if result.Error != nil {
		size += len(result.Error.Body) + len(result.Error.Message) + len(result.Error.Code)
	}
	if size > maxDataBytes {
		return fmt.Errorf("plugin response and error exceed 32 MiB")
	}
	return nil
}

type pluginRPC interface {
	GetManifest(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
	Execute(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
}

type httpRPC interface {
	Do(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
}

// These descriptors use the standard protobuf BytesValue message. The payload
// schemas above are the versioned SDK contract; no custom gRPC codec is needed.
var pluginDescriptor = grpc.ServiceDesc{
	ServiceName: pluginService,
	HandlerType: (*pluginRPC)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetManifest", Handler: manifestHandler},
		{MethodName: "Execute", Handler: executeHandler},
	},
}

var httpDescriptor = grpc.ServiceDesc{
	ServiceName: httpService,
	HandlerType: (*httpRPC)(nil),
	Methods:     []grpc.MethodDesc{{MethodName: "Do", Handler: httpHandler}},
}

//nolint:revive // gRPC's MethodHandler requires server before context.
func manifestHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return handle(ctx, server, decode, interceptor, pluginService+"/GetManifest", func(ctx context.Context, message *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
		implementation, ok := server.(pluginRPC)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid plugin RPC implementation")
		}
		return implementation.GetManifest(ctx, message)
	})
}

//nolint:revive // gRPC's MethodHandler requires server before context.
func executeHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return handle(ctx, server, decode, interceptor, pluginService+"/Execute", func(ctx context.Context, message *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
		implementation, ok := server.(pluginRPC)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid plugin RPC implementation")
		}
		return implementation.Execute(ctx, message)
	})
}

//nolint:revive // gRPC's MethodHandler requires server before context.
func httpHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return handle(ctx, server, decode, interceptor, httpService+"/Do", func(ctx context.Context, message *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
		implementation, ok := server.(httpRPC)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid HTTP RPC implementation")
		}
		return implementation.Do(ctx, message)
	})
}

func handle(ctx context.Context, server any, decode func(any) error, interceptor grpc.UnaryServerInterceptor, method string, call func(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)) (any, error) {
	message := new(wrapperspb.BytesValue)
	if err := decode(message); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return call(ctx, message)
	}
	return interceptor(ctx, message, &grpc.UnaryServerInfo{Server: server, FullMethod: "/" + method}, func(ctx context.Context, request any) (any, error) {
		message, ok := request.(*wrapperspb.BytesValue)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid plugin RPC request")
		}
		return call(ctx, message)
	})
}

func invoke(ctx context.Context, conn *grpc.ClientConn, method string, request, result any) error {
	message, err := pack(request)
	if err != nil {
		return err
	}
	var response wrapperspb.BytesValue
	if err := conn.Invoke(ctx, "/"+method, message, &response, grpc.MaxCallRecvMsgSize(maxWireBytes), grpc.MaxCallSendMsgSize(maxWireBytes)); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("plugin RPC: %w", err)
	}
	return unpack(&response, result)
}
