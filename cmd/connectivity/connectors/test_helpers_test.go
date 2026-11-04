package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

type connectorTransportMock struct {
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

func (m connectorTransportMock) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	query := request.URL.Query()
	pageSize, _ := strconv.ParseInt(query.Get("pageSize"), 10, 32)
	options := connectivityinternal.ListOptions{PageSize: int32(pageSize), Cursor: query.Get("cursor"), Query: query.Get("query")}
	path := strings.TrimPrefix(request.URL.Path, "/api/connectivity/connectors")
	var data any
	var err error
	switch {
	case path == "":
		var page *connectorPage
		page, err = m.list(ctx, options)
		if page != nil {
			data = connectivity.ConnectorCursor{Cursor: connectivity.ConnectorCursorCursor{Data: page.Items, PageSize: int(page.PageSize), HasMore: page.HasMore, Next: connectivityinternal.OptionalString(page.Next)}}
		}
	case path == "/_facets":
		var facets *connectivity.FacetDistribution
		facets, err = m.facets(ctx, options.Query)
		if facets != nil {
			data = facets
		}
	case strings.HasSuffix(path, "/versions"):
		name := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/versions")
		page := &connectorVersionPage{Items: []connectivity.ConnectorVersionSummary{}}
		if m.listVersions != nil {
			page, err = m.listVersions(ctx, name, options)
		}
		if page != nil {
			data = connectivity.ConnectorVersionCursor{Cursor: connectivity.ConnectorVersionCursorCursor{Data: page.Items, PageSize: int(page.PageSize), HasMore: page.HasMore, Next: connectivityinternal.OptionalString(page.Next)}}
		}
	case strings.Contains(path, "/versions/"):
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/versions/", 2)
		version := &connectivity.ConnectorVersion{Version: parts[1]}
		if m.getVersion != nil {
			version, err = m.getVersion(ctx, parts[0], parts[1])
		}
		if version != nil {
			data = version
		}
	default:
		var connector *connectivity.Connector
		connector, err = m.get(ctx, strings.TrimPrefix(path, "/"))
		if connector != nil {
			data = connector
		}
	}
	var apiErr *sdkerrors.SDKError
	if errors.As(err, &apiErr) {
		return connectorResponse(apiErr.StatusCode, apiErr.Body), nil
	}
	if err != nil {
		return nil, err
	}
	if data == nil {
		return connectorResponse(http.StatusOK, ""), nil
	}
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return connectorResponse(http.StatusOK, string(body)), nil
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

func sdkFor(transport connectorTransportMock) *formance.FormanceV1 {
	return formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: transport})).Connectivity.V1
}

func testNewListCommand(client *formance.FormanceV1) *cobra.Command {
	controller := NewListController()
	command := NewListCommand()
	command.RunE = func(cmd *cobra.Command, args []string) error {
		renderable, err := controller.run(cmd, args, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, renderable)
	}
	return command
}

func testNewShowCommand(client *formance.FormanceV1) *cobra.Command {
	controller := NewShowController()
	command := NewShowCommand()
	command.RunE = func(cmd *cobra.Command, args []string) error {
		renderable, err := controller.run(cmd, args, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, renderable)
	}
	return command
}

func testNewFacetsCommand(client *formance.FormanceV1) *cobra.Command {
	controller := NewFacetsController()
	command := NewFacetsCommand()
	command.RunE = func(cmd *cobra.Command, args []string) error {
		renderable, err := controller.run(cmd, args, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, renderable)
	}
	return command
}

func testCompleteConnectorNames(client *formance.FormanceV1) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if cmd.Context() == nil {
			cmd.SetContext(context.Background())
		}
		return completeConnectorNames(cmd, client, toComplete)
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
