package connectors

import (
	"bytes"
	"context"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type connectorClientMock struct {
	connectivityclient.Client
	list         func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error)
	get          func(context.Context, string) (*connectivityclient.Connector, error)
	listVersions func(context.Context, string) (*connectivityclient.ConnectorVersionList, error)
	getVersion   func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error)
}

func (m connectorClientMock) ListConnectors(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
	return m.list(ctx, options)
}

func (m connectorClientMock) GetConnector(ctx context.Context, name string) (*connectivityclient.Connector, error) {
	return m.get(ctx, name)
}

func (m connectorClientMock) ListConnectorVersions(ctx context.Context, connector string) (*connectivityclient.ConnectorVersionList, error) {
	if m.listVersions == nil {
		return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{}}, nil
	}
	return m.listVersions(ctx, connector)
}

func (m connectorClientMock) GetConnectorVersion(ctx context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
	if m.getVersion == nil {
		return &connectivityclient.ConnectorVersion{Version: version, Image: "registry/connector:" + version}, nil
	}
	return m.getVersion(ctx, connector, version)
}

func connectorFixture(name string) connectivityclient.Connector {
	return connectivityclient.Connector{
		Metadata: connectivityclient.ObjectMeta{Name: fctl.Ptr(name)},
		Spec: connectivityclient.ConnectorSpec{
			DisplayName: fctl.Ptr("Connector display name"),
			Description: fctl.Ptr("Connector description"),
			ImageURL:    fctl.Ptr("registry/connector"),
			Catalog:     fctl.Ptr("public"),
			Tags:        []string{"payments", "webhooks"},
		},
		Status: &connectivityclient.ConnectorStatus{Phase: fctl.Ptr("Ready")},
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
