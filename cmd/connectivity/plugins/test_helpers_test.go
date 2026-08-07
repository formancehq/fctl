package plugins

import (
	"bytes"
	"context"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type pluginClientMock struct {
	connectivityclient.Client
	list func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error)
	get  func(context.Context, string) (*connectivityclient.Plugin, error)
}

func (m pluginClientMock) ListPlugins(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
	return m.list(ctx, options)
}

func (m pluginClientMock) GetPlugin(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
	return m.get(ctx, name)
}

func pluginFixture(name string) connectivityclient.Plugin {
	return connectivityclient.Plugin{
		Metadata: connectivityclient.ObjectMeta{Name: fctl.Ptr(name)},
		Spec: connectivityclient.PluginSpec{
			Image:          "registry/plugin",
			Description:    fctl.Ptr("Plugin description"),
			DefaultVersion: fctl.Ptr("2.0.0"),
			Capabilities:   []string{"payments", "webhooks"},
		},
		Status: &connectivityclient.PluginStatus{Phase: fctl.Ptr("Ready")},
	}
}

func factoryReturning(client connectivityclient.Client) connectivityinternal.ClientFactory {
	return func(*cobra.Command) (connectivityclient.Client, error) {
		return client, nil
	}
}

func executeCommand(command *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}
