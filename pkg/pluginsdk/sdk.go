// Package pluginsdk provides the fctl plugin SDK.
//
// It integrates HashiCorp go-plugin with gRPC to allow external binaries
// to provide fctl commands. Both the core (client side) and plugins (server side)
// use this package.
package pluginsdk

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
)

// HandshakeConfig is the shared handshake that both the core and plugins
// must agree on. Changing this breaks compatibility.
var HandshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FCTL_PLUGIN",
	MagicCookieValue: "formance",
}

// FctlPlugin is the interface that every plugin must implement.
type FctlPlugin interface {
	GetManifest(ctx context.Context) (*pluginpb.PluginManifest, error)
	Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error)
}

// PluginMap is the map of plugin types that go-plugin uses during negotiation.
var PluginMap = map[string]goplugin.Plugin{
	"fctl-plugin": &GRPCPlugin{},
}

// GRPCPlugin implements goplugin.GRPCPlugin for the fctl plugin protocol.
type GRPCPlugin struct {
	goplugin.Plugin

	// Impl is only set on the server (plugin) side.
	Impl FctlPlugin
}

func (p *GRPCPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginpb.RegisterPluginServiceServer(s, &grpcServer{impl: p.Impl})
	return nil
}

func (p *GRPCPlugin) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{client: pluginpb.NewPluginServiceClient(c)}, nil
}

// grpcServer wraps an FctlPlugin implementation as a gRPC server.
type grpcServer struct {
	pluginpb.UnimplementedPluginServiceServer
	impl FctlPlugin
}

func (s *grpcServer) GetManifest(ctx context.Context, req *pluginpb.GetManifestRequest) (*pluginpb.GetManifestResponse, error) {
	manifest, err := s.impl.GetManifest(ctx)
	if err != nil {
		return nil, err
	}
	return &pluginpb.GetManifestResponse{Manifest: manifest}, nil
}

func (s *grpcServer) Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	return s.impl.Execute(ctx, req)
}

// grpcClient wraps a gRPC client connection as an FctlPlugin.
type grpcClient struct {
	client pluginpb.PluginServiceClient
}

func (c *grpcClient) GetManifest(ctx context.Context) (*pluginpb.PluginManifest, error) {
	resp, err := c.client.GetManifest(ctx, &pluginpb.GetManifestRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Manifest, nil
}

func (c *grpcClient) Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	return c.client.Execute(ctx, req)
}

// Serve is the entry point for plugins. Call this from your main() function.
//
//	func main() {
//	    pluginsdk.Serve(&MyPlugin{})
//	}
func Serve(impl FctlPlugin) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]goplugin.Plugin{
			"fctl-plugin": &GRPCPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
