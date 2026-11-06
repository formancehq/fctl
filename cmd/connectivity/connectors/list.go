package connectors

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
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
		connectivityinternal.CompleteFilterExpressions(connectivityinternal.ResourceConnectors),
	); err != nil {
		panic(err)
	}
	return command
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
	if response == nil {
		return nil, fmt.Errorf("list connectivity connectors: empty response")
	}

	if response.ConnectorCursor == nil {
		return nil, fmt.Errorf("list connectivity connectors: empty response")
	}
	page := response.ConnectorCursor.Cursor
	if page.Data == nil {
		return nil, fmt.Errorf("list connectivity connectors: cursor.data must be an array")
	}
	c.store.Connectors = page.Data
	c.store.Cursor = cursorFromList(page.PageSize, page.HasMore, page.Next)
	return c, nil
}

func cursorFromList(pageSize int, hasMore bool, next *string) fctl.Cursor {
	return fctl.Cursor{PageSize: int64(pageSize), HasMore: hasMore, Next: connectivityinternal.OptionalString(stringValue(next))}
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	rows := fctl.Map(c.store.Connectors, func(connector connectivity.Connector) []string {
		return []string{
			connector.Name,
			stringValue(connector.DisplayName),
			strings.Join(connector.Tags, ", "),
		}
	})
	// No description column: descriptions are full sentences, and pterm sizes
	// columns to their content, so one connector wrapped every row and the
	// table stopped lining up. `connectors show` prints the description.
	rows = fctl.Prepend(rows, []string{"Name", "Display Name", "Tags"})
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
