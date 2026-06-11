package schemas

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/ledger"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"

	internal "github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Schemas []ledger.V2SchemaData `json:"schemas"`
	Cursor  fctl.Cursor           `json:"cursor"`
}
type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		Schemas: []ledger.V2SchemaData{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List all schemas for a ledger"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ListStore](c),
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

	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}

	req := operations.V2ListSchemasRequest{
		Ledger: fctl.GetString(cmd, internal.LedgerFlag),
	}
	if cursor != "" {
		req.Cursor = fctl.Ptr(cursor)
	} else {
		req.PageSize = fctl.Ptr(int64(pageSize))
	}

	response, err := stackClient.Ledger.V2.ListSchemas(cmd.Context(), req)
	if err != nil {
		return nil, err
	}

	cur := response.V2SchemasCursorResponse.V2SchemasCursor
	c.store.Schemas = cur.Data
	c.store.Cursor = fctl.Cursor{HasMore: cur.HasMore, PageSize: cur.PageSize, Next: cur.Next, Previous: cur.Previous}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	tableData := fctl.Map(c.store.Schemas, func(schema ledger.V2SchemaData) []string {
		return []string{
			schema.Version,
			schema.CreatedAt.Format(time.RFC3339),
			fmt.Sprintf("%d", len(schema.V2ChartOfAccounts)),
			fmt.Sprintf("%d", len(schema.V2QueryTemplates)),
			fmt.Sprintf("%d", len(schema.V2TransactionTemplates)),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"Version", "Created at", "Chart", "Queries", "Transactions"})

	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}
