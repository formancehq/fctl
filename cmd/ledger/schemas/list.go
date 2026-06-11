package schemas

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/ledger"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/go-libs/v4/pointer"

	internal "github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Schemas []ledger.V2SchemaData `json:"schemas"`
}
type ListController struct {
	store        *ListStore
	pageSizeFlag string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		Schemas: []ledger.V2SchemaData{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store:        NewDefaultListStore(),
		pageSizeFlag: "page-size",
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List all schemas for a ledger"),
		fctl.WithIntFlag(c.pageSizeFlag, 15, "Page size"),
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

	response, err := stackClient.Ledger.V2.ListSchemas(cmd.Context(), operations.V2ListSchemasRequest{
		Ledger:   fctl.GetString(cmd, internal.LedgerFlag),
		PageSize: pointer.For(int64(fctl.GetInt(cmd, c.pageSizeFlag))),
	})
	if err != nil {
		return nil, err
	}

	c.store.Schemas = response.V2SchemasCursorResponse.V2SchemasCursor.Data
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	if len(c.store.Schemas) == 0 {
		fctl.Println("No schemas found.")
		return nil
	}

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

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
