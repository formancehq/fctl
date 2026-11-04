package connectors

import (
	"bytes"
	"context"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type connectorClientMock struct {
	connectivityinternal.API
	list         func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error)
	facets       func(context.Context, string) (*connectivity.FacetDistribution, error)
	get          func(context.Context, string) (*connectivity.Connector, error)
	listVersions func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error)
	getVersion   func(context.Context, string, string) (*connectivity.ConnectorVersion, error)
}

type connectorPage struct {
	Items    []connectivity.Connector
	PageSize int32
	HasMore  bool
	Next     string
}

type connectorVersionPage struct {
	Items    []connectivity.ConnectorVersionSummary
	PageSize int32
	HasMore  bool
	Next     string
}

func (m connectorClientMock) ListConnectors(ctx context.Context, request operations.ListConnectorsRequest, _ *operations.ListConnectorsSecurity, _ ...operations.Option) (*operations.ListConnectorsResponse, error) {
	page, err := m.list(ctx, mockListOptions(request.PageSize, request.Cursor, request.Query))
	if err != nil || page == nil {
		return nil, err
	}
	return &operations.ListConnectorsResponse{StatusCode: 200, ConnectorCursor: &connectivity.ConnectorCursor{Cursor: connectivity.ConnectorCursorCursor{Data: page.Items, PageSize: int(page.PageSize), HasMore: page.HasMore, Next: connectivityinternal.OptionalString(page.Next)}}}, nil
}

func (m connectorClientMock) GetConnectorFacets(ctx context.Context, request operations.GetConnectorFacetsRequest, _ *operations.GetConnectorFacetsSecurity, _ ...operations.Option) (*operations.GetConnectorFacetsResponse, error) {
	facets, err := m.facets(ctx, stringValue(request.Query))
	return &operations.GetConnectorFacetsResponse{StatusCode: 200, FacetDistribution: facets}, err
}

func (m connectorClientMock) GetConnector(ctx context.Context, request operations.GetConnectorRequest, _ *operations.GetConnectorSecurity, _ ...operations.Option) (*operations.GetConnectorResponse, error) {
	connector, err := m.get(ctx, request.Name)
	return &operations.GetConnectorResponse{StatusCode: 200, Connector: connector}, err
}

func (m connectorClientMock) ListConnectorVersions(ctx context.Context, request operations.ListConnectorVersionsRequest, _ *operations.ListConnectorVersionsSecurity, _ ...operations.Option) (*operations.ListConnectorVersionsResponse, error) {
	page := &connectorVersionPage{Items: []connectivity.ConnectorVersionSummary{}}
	var err error
	if m.listVersions != nil {
		page, err = m.listVersions(ctx, request.Name, mockListOptions(request.PageSize, request.Cursor, request.Query))
	}
	if err != nil || page == nil {
		return nil, err
	}
	return &operations.ListConnectorVersionsResponse{StatusCode: 200, ConnectorVersionCursor: &connectivity.ConnectorVersionCursor{Cursor: connectivity.ConnectorVersionCursorCursor{Data: page.Items, PageSize: int(page.PageSize), HasMore: page.HasMore, Next: connectivityinternal.OptionalString(page.Next)}}}, nil
}

func (m connectorClientMock) GetConnectorVersion(ctx context.Context, request operations.GetConnectorVersionRequest, _ *operations.GetConnectorVersionSecurity, _ ...operations.Option) (*operations.GetConnectorVersionResponse, error) {
	version := &connectivity.ConnectorVersion{Version: request.Version}
	var err error
	if m.getVersion != nil {
		version, err = m.getVersion(ctx, request.Name, request.Version)
	}
	return &operations.GetConnectorVersionResponse{StatusCode: 200, ConnectorVersion: version}, err
}

func mockListOptions(pageSize *int, cursor, query *string) connectivityinternal.ListOptions {
	options := connectivityinternal.ListOptions{Cursor: stringValue(cursor), Query: stringValue(query)}
	if pageSize != nil {
		options.PageSize = int32(*pageSize)
	}
	return options
}

func connectorFixture(name string) connectivity.Connector {
	return connectivity.Connector{
		Name:              name,
		DisplayName:       fctl.Ptr("Connector display name"),
		Description:       fctl.Ptr("Connector description"),
		Catalog:           fctl.Ptr("public"),
		Tags:              []string{"payments", "webhooks"},
		Tagline:           fctl.Ptr("Payments infrastructure"),
		LatestVersion:     fctl.Ptr("2.0.0"),
		ConnectorBranding: &connectivity.ConnectorBranding{DisplayName: fctl.Ptr("Brand display name")},
	}
}

func factoryReturning(client connectivityinternal.API) connectivityinternal.ClientFactory {
	return func(*cobra.Command) (connectivityinternal.API, error) {
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
