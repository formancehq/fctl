// Package transport serves and loads external fctl plugins over local gRPC.
// The host retains its HTTP client and credentials; plugins receive a scoped
// HTTP callback instead. Executables must be trusted: this is not a sandbox.
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

// ErrClosed is returned when a terminated runtime is used again.
var ErrClosed = errors.New("plugin runtime is closed")

// Client is a running external plugin. It implements the same contract as an
// embedded plugin. Close releases the process and all its HTTP callbacks.
type Client struct {
	remote   *remotePlugin
	process  *goplugin.Client
	http     *http.Client
	endpoint *url.URL
	lifetime context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	stop     func() bool
}

var _ pluginsdk.Plugin = (*Client)(nil)

// Open starts binary without arguments or inherited environment variables.
// Cancellation of ctx, or of an RPC, terminates the process. A nil client and
// empty endpoint are allowed for metadata inspection; Execute needs both.
// The host must verify the binary's provenance/digest before calling Open.
func Open(ctx context.Context, binary string, client *http.Client, endpoint string) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(binary) {
		return nil, fmt.Errorf("plugin executable must be an absolute path")
	}
	target, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithCancel(ctx)
	command := exec.CommandContext(lifetime, binary)
	command.Env = []string{}
	command.WaitDelay = time.Second
	if runtime.GOOS == "windows" {
		// Required by the Windows runtime; no user configuration is inherited.
		command.Env = append(command.Env, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"))
	}
	process := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: handshake,
		Plugins:         goplugin.PluginSet{pluginName: &grpcPlugin{}},
		Cmd:             command,
		SkipHostEnv:     true,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		StartTimeout:    10 * time.Second,
		AutoMTLS:        true,
		Logger:          hclog.NewNullLogger(),
		SyncStdout:      io.Discard,
		SyncStderr:      io.Discard,
		Stderr:          io.Discard,
		GRPCDialOptions: []grpc.DialOption{grpc.WithDisableRetry()},
	})
	protocol, err := process.Client()
	if err != nil {
		cancel()
		process.Kill()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("start external plugin: %w", err)
	}
	dispensed, err := protocol.Dispense(pluginName)
	if err != nil {
		cancel()
		process.Kill()
		return nil, fmt.Errorf("load external plugin: %w", err)
	}
	remote, ok := dispensed.(*remotePlugin)
	if !ok {
		cancel()
		process.Kill()
		return nil, fmt.Errorf("invalid external plugin implementation")
	}
	result := &Client{remote: remote, process: process, http: client, endpoint: target,
		lifetime: lifetime, cancel: cancel, done: make(chan struct{})}
	result.mu.Lock()
	result.stop = context.AfterFunc(ctx, result.close)
	result.mu.Unlock()
	if ctx.Err() != nil {
		result.close()
		return nil, ctx.Err()
	}
	return result, nil
}

func parseEndpoint(endpoint string) (*url.URL, error) {
	if endpoint == "" {
		return nil, nil
	}
	if err := httpclient.ValidateURL(endpoint); err != nil {
		return nil, err
	}
	target, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return nil, err
	}
	if err := validatePath(target); err != nil {
		return nil, err
	}
	return target, nil
}

// Close is safe to call concurrently and more than once. It waits until the
// subprocess is reaped. A closed or canceled Client cannot be restarted.
func (c *Client) Close() error {
	c.close()
	return nil
}

func (c *Client) close() {
	c.once.Do(func() {
		c.mu.Lock()
		if c.stop != nil {
			c.stop()
		}
		c.mu.Unlock()
		c.cancel()
		c.process.Kill()
		close(c.done)
	})
}

func (c *Client) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.lifetime.Err() != nil {
		return ErrClosed
	}
	return nil
}

// GetManifest uses factory(nil) in the plugin. It never enables HTTP callbacks.
func (c *Client) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := c.check(ctx); err != nil {
		return pluginsdk.Manifest{}, err
	}
	stop := context.AfterFunc(ctx, c.close)
	defer stop()
	var manifest pluginsdk.Manifest
	if err := invoke(ctx, c.remote.conn, pluginService+"/GetManifest", struct{}{}, &manifest); err != nil {
		return pluginsdk.Manifest{}, c.rpcError(ctx, err)
	}
	if manifest.ProtocolVersion != pluginsdk.ProtocolVersion {
		return pluginsdk.Manifest{}, fmt.Errorf("unsupported plugin protocol version %d", manifest.ProtocolVersion)
	}
	return manifest, nil
}

// Execute preserves SDK requests and partial results. The HTTP callback is
// unique to this invocation, and is removed even when execution fails.
func (c *Client) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	if err := c.check(ctx); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if c.http == nil || c.endpoint == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("plugin execution requires a host HTTP client and endpoint")
	}
	if request.Endpoint == "" {
		request.Endpoint = c.endpoint.String()
	}
	if strings.TrimRight(request.Endpoint, "/") != c.endpoint.String() {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("plugin endpoint differs from host endpoint")
	}
	if err := validateRequest(request); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	stop := context.AfterFunc(ctx, c.close)
	defer stop()
	callback := c.serveHTTP()
	defer callback.stop()
	var response resultWire
	err := invoke(ctx, c.remote.conn, pluginService+"/Execute", executeWire{BrokerID: callback.id, Request: toWire(request)}, &response)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, c.rpcError(ctx, err)
	}
	if err := validateResult(response); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	return pluginsdk.ExecuteResponse{Data: response.Data}, response.Error.err()
}

func (c *Client) rpcError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		c.close()
		return ctx.Err()
	}
	if c.lifetime.Err() != nil {
		return ErrClosed
	}
	// The peer can report cancellation before the local context timer fires.
	// Both transport cancellation statuses must terminate and reap the plugin.
	code := status.Code(err)
	if code == codes.DeadlineExceeded || code == codes.Canceled {
		c.close()
		if code == codes.DeadlineExceeded {
			return context.DeadlineExceeded
		}
		return context.Canceled
	}
	return err
}

func validateRequest(request pluginsdk.ExecuteRequest) error {
	if len(request.Body) > maxBodyBytes {
		return fmt.Errorf("plugin request body exceeds 4 MiB")
	}
	metadata := toWire(request)
	metadata.Body = nil
	encoded, err := pack(metadata)
	if err != nil || len(encoded.Value) > maxMetaBytes {
		return fmt.Errorf("plugin request metadata exceeds 1 MiB")
	}
	for key := range request.Context {
		if credentialKey(key) {
			return fmt.Errorf("plugin context must not contain credentials")
		}
	}
	return nil
}
