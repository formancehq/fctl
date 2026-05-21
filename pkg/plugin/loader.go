package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk"
	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
	goplugin "github.com/hashicorp/go-plugin"
)

// LoadedPlugin represents a plugin that has been loaded and is ready to use.
type LoadedPlugin struct {
	Name           string
	Version        string
	CompatibleWith string
	Client         pluginsdk.FctlPlugin
	client         *goplugin.Client
	stderr         *CapturedWriter
}

// CapturedStderr returns the captured stderr content from the plugin process.
func (l *LoadedPlugin) CapturedStderr() string {
	if l.stderr != nil {
		return l.stderr.String()
	}
	return ""
}

// Killable is implemented by plugin clients that manage a process.
type Killable interface {
	Kill()
}

// Kill terminates the plugin process.
func (l *LoadedPlugin) Kill() {
	if k, ok := l.Client.(Killable); ok {
		k.Kill()
	}
	if l.client != nil {
		l.client.Kill()
	}
}

// LoadPlugin starts a plugin binary and returns a LoadedPlugin.
// If debug is true, plugin stderr is prefixed and forwarded to os.Stderr.
func LoadPlugin(name, binaryPath string, opts ...LoadPluginOption) (*LoadedPlugin, error) {
	var cfg loadPluginConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	captured, stderrWriter := NewPluginStderr(name, cfg.debug)

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: pluginsdk.HandshakeConfig,
		Plugins:         pluginsdk.PluginMap,
		Cmd:             exec.Command(binaryPath),
		Stderr:          stderrWriter,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin %s: %w", name, err)
	}

	raw, err := rpcClient.Dispense("fctl-plugin")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin %s: %w", name, err)
	}

	fctlPlugin, ok := raw.(pluginsdk.FctlPlugin)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin %s does not implement FctlPlugin interface", name)
	}

	return &LoadedPlugin{
		Name:   name,
		Client: fctlPlugin,
		client: client,
		stderr: captured,
	}, nil
}

type loadPluginConfig struct {
	debug bool
}

// LoadPluginOption configures LoadPlugin behavior.
type LoadPluginOption func(*loadPluginConfig)

// WithDebug enables debug mode for the plugin (stderr prefixing).
func WithDebug() LoadPluginOption {
	return func(c *loadPluginConfig) {
		c.debug = true
	}
}

// LazyPluginClient wraps a cached manifest and defers process spawn to first Execute call.
// GetManifest returns the cached manifest without spawning a process.
type LazyPluginClient struct {
	manifest   *pluginpb.PluginManifest
	name       string
	binaryPath string
	debug      bool

	mu     sync.Mutex
	inner  pluginsdk.FctlPlugin
	client *goplugin.Client
}

// NewLazyPluginClient creates a lazy client that serves GetManifest from cache
// and only spawns the plugin process on Execute.
func NewLazyPluginClient(name, binaryPath string, manifest *pluginpb.PluginManifest, debug bool) *LazyPluginClient {
	return &LazyPluginClient{
		manifest:   manifest,
		name:       name,
		binaryPath: binaryPath,
		debug:      debug,
	}
}

func (l *LazyPluginClient) GetManifest(_ context.Context) (*pluginpb.PluginManifest, error) {
	return l.manifest, nil
}

func (l *LazyPluginClient) Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	inner, err := l.ensureLoaded()
	if err != nil {
		return nil, err
	}
	return inner.Execute(ctx, req)
}

func (l *LazyPluginClient) ensureLoaded() (pluginsdk.FctlPlugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.inner != nil {
		return l.inner, nil
	}

	var opts []LoadPluginOption
	if l.debug {
		opts = append(opts, WithDebug())
	}
	loaded, err := LoadPlugin(l.name, l.binaryPath, opts...)
	if err != nil {
		return nil, err
	}

	l.inner = loaded.Client
	l.client = loaded.client
	return l.inner, nil
}

// Kill terminates the underlying plugin process if it was spawned.
func (l *LazyPluginClient) Kill() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.client != nil {
		l.client.Kill()
		l.client = nil
		l.inner = nil
	}
}
