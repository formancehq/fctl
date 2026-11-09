package connectors

import (
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ListStore struct {
	Connectors []connectivity.Connector `json:"connectors"`
	Cursor     fctl.Cursor              `json:"cursor"`
}

type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController() *ListController {
	return &ListController{
		store: &ListStore{Connectors: []connectivity.Connector{}},
	}
}

func NewListCommand() *cobra.Command {
	controller := NewListController()
	return fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List catalog connectors"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		connectivityinternal.WithListQueryFlags(),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController[*ListStore](controller),
		fctl.WithExamples(
			"fctl connectivity connectors list --filter tags=provider:psp",
			"fctl connectivity connectors list --filter 'name~%stripe%' --page-size 50",
		),
		fctl.WithoutFileCompletion(connectivityinternal.QueryFlag, "cursor", "page-size"),
		fctl.WithFlagCompletion(connectivityinternal.FilterFlag, connectivityinternal.CompleteFilterExpressions(connectivityinternal.ResourceConnectors)),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, nil, stackClient.Connectivity.V1)
}

func (c *ListController) run(cmd *cobra.Command, _ []string, client *formance.FormanceV1) (fctl.Renderable, error) {
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	query, err := connectivityinternal.GetListQuery(cmd)
	if err != nil {
		return nil, err
	}

	response, err := client.ListConnectors(cmd.Context(), operations.ListConnectorsRequest{
		PageSize: fctl.Ptr(int(pageSize)),
		Cursor:   connectivityinternal.OptionalString(cursor),
		Query:    connectivityinternal.OptionalString(query),
	}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil || response.ConnectorCursor == nil {
		return nil, fmt.Errorf("list connectivity connectors: empty response")
	}
	page := response.ConnectorCursor.Cursor
	if page.Data == nil {
		return nil, fmt.Errorf("list connectivity connectors: cursor.data must be an array")
	}
	c.store.Connectors = page.Data
	c.store.Cursor = fctl.Cursor{PageSize: int64(page.PageSize), HasMore: page.HasMore, Next: connectivityinternal.OptionalString(fctl.StringPointerToString(page.Next))}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	// Display names repeat the connector name, so the list shows the tagline
	// instead; connectors show keeps every field.
	rows := fctl.Map(c.store.Connectors, func(connector connectivity.Connector) []string {
		return []string{connector.Name, fctl.StringPointerToString(connector.LatestVersion), fctl.StringPointerToString(connector.Tagline)}
	})
	columns := []fctl.TableColumn{{}, {}, {Fit: fctl.FitCut, Min: 12}}
	if err := fctl.RenderTable(out, []string{"Name", "Latest Version", "Tagline"}, columns, rows); err != nil {
		return err
	}
	return fctl.RenderNextPage(out, c.store.Cursor)
}
