package connectors

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Connectors []connectivityclient.Connector `json:"connectors"`
	Cursor     fctl.Cursor                    `json:"cursor"`
}

type ListController struct {
	factory connectivityinternal.ClientFactory
	store   *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController(factory connectivityinternal.ClientFactory) *ListController {
	return &ListController{
		factory: factory,
		store:   &ListStore{Connectors: []connectivityclient.Connector{}},
	}
}

func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewListController(factory)
	command := fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List available Connectivity connectors"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		connectivityinternal.WithListQueryFlags(),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController[*ListStore](controller),
	)
	if err := command.RegisterFlagCompletionFunc(
		connectivityinternal.FilterFlag,
		connectivityinternal.CompleteFilterExpressions(factory, connectivityclient.ResourceConnectors),
	); err != nil {
		panic(err)
	}
	return command
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
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

	response, err := client.ListConnectors(cmd.Context(), connectivityclient.ListOptions{
		PageSize: pageSize,
		Cursor:   cursor,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("list connectivity connectors: empty response")
	}

	c.store.Connectors = response.Items
	c.store.Cursor = cursorFromList(response.PageSize, response.HasMore, response.Next)
	return c, nil
}

func cursorFromList(pageSize int32, hasMore bool, next string) fctl.Cursor {
	cursor := fctl.Cursor{PageSize: int64(pageSize), HasMore: hasMore}
	if next != "" {
		cursor.Next = fctl.Ptr(next)
	}
	return cursor
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	rows := fctl.Map(c.store.Connectors, func(connector connectivityclient.Connector) []string {
		return []string{
			stringValue(connector.Metadata.Name),
			stringValue(connector.Spec.DisplayName),
			stringValue(connector.Spec.Description),
			strings.Join(connector.Spec.Tags, ", "),
			connectorPhase(connector.Status),
		}
	})
	rows = fctl.Prepend(rows, []string{"Name", "Display Name", "Description", "Tags", "Phase"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(rows).
		Render(); err != nil {
		return err
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func connectorPhase(status *connectivityclient.ConnectorStatus) string {
	if status == nil {
		return ""
	}
	return stringValue(status.Phase)
}
