package plugin

import (
	"fmt"
	"os/exec"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	goplugin "github.com/hashicorp/go-plugin"
)

// LoadedPlugin represents a plugin that has been loaded and is ready to use.
type LoadedPlugin struct {
	Name    string
	Version string
	Client  pluginsdk.FctlPlugin
	client  *goplugin.Client
}

// Kill terminates the plugin process.
func (l *LoadedPlugin) Kill() {
	if l.client != nil {
		l.client.Kill()
	}
}

// LoadPlugin starts a plugin binary and returns a LoadedPlugin.
func LoadPlugin(name, binaryPath string) (*LoadedPlugin, error) {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: pluginsdk.HandshakeConfig,
		Plugins:         pluginsdk.PluginMap,
		Cmd:             exec.Command(binaryPath),
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
	}, nil
}
