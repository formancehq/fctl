package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

const pluginName = "fctl-plugin"

// The transport handshake is intentionally separate from the SDK's manifest
// protocol. Its cookie identifies compatible executables, not trust in them.
var handshake = goplugin.HandshakeConfig{
	ProtocolVersion: 1, MagicCookieKey: "FCTL_PLUGIN_V4", MagicCookieValue: "formance-fctl-v4",
}

type grpcPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	factory func(*http.Client) pluginsdk.Plugin
}

func (p *grpcPlugin) GRPCServer(broker *goplugin.GRPCBroker, server *grpc.Server) error {
	server.RegisterService(&pluginDescriptor, &pluginServer{factory: p.factory, broker: broker})
	return nil
}

func (*grpcPlugin) GRPCClient(_ context.Context, broker *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return &remotePlugin{broker: broker, conn: conn}, nil
}

// Serve runs a plugin executable. Call it from main with the same factory used
// by an embedded plugin. Metadata is requested from factory(nil), without HTTP.
// Execution receives a client which calls back into the host for every request.
func Serve(factory func(*http.Client) pluginsdk.Plugin) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake,
		Plugins:         goplugin.PluginSet{pluginName: &grpcPlugin{factory: factory}},
		Logger:          hclog.NewNullLogger(),
		GRPCServer: func(options []grpc.ServerOption) *grpc.Server {
			return grpc.NewServer(append(options, grpc.MaxRecvMsgSize(maxWireBytes), grpc.MaxSendMsgSize(maxWireBytes))...)
		},
	})
}

type remotePlugin struct {
	broker *goplugin.GRPCBroker
	conn   *grpc.ClientConn
}

type pluginServer struct {
	factory func(*http.Client) pluginsdk.Plugin
	broker  *goplugin.GRPCBroker
}

func (s *pluginServer) GetManifest(ctx context.Context, _ *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
	manifest, err := s.factory(nil).GetManifest(ctx)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	message, err := pack(manifest)
	if err == nil && len(message.Value) > maxMetaBytes {
		return nil, status.Error(codes.ResourceExhausted, "plugin manifest exceeds 1 MiB")
	}
	return message, err
}

func (s *pluginServer) Execute(ctx context.Context, message *wrapperspb.BytesValue) (response *wrapperspb.BytesValue, err error) {
	var request executeWire
	if err := unpack(message, &request); err != nil {
		return nil, err
	}
	if request.BrokerID == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing host HTTP callback")
	}
	conn, err := s.broker.DialWithOptions(request.BrokerID, grpc.WithDisableRetry())
	if err != nil {
		return nil, status.Error(codes.Unavailable, "connect host HTTP callback")
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	client := &http.Client{Transport: &callbackTransport{conn: conn, execution: ctx}, CheckRedirect: func(*http.Request, []*http.Request) error {
		// The host client owns redirects, and must check each hop itself.
		return http.ErrUseLastResponse
	}}
	result, executeErr := s.factory(client).Execute(ctx, request.Request.request())
	resultMessage := resultWire{Data: result.Data, Error: encodeError(executeErr)}
	if err := validateResult(resultMessage); err != nil {
		return nil, status.Error(codes.ResourceExhausted, fmt.Sprintf("invalid plugin result: %s", err))
	}
	return pack(resultMessage)
}
