package accounts

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/pkg"
)

type ListStore struct {
	Accounts []shared.V2Account `json:"accounts"`
}
type ListController struct {
	store        *ListStore
	metadataFlag string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store:        NewDefaultListStore(),
		metadataFlag: "metadata",
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List accounts"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithStringSliceFlag(c.metadataFlag, []string{}, "Filter accounts with metadata"),
		fctl.WithController[*ListStore](c),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, c.metadataFlag))
	if err != nil {
		return nil, err
	}

	body := make([]map[string]map[string]any, 0)
	for key, value := range metadata {
		body = append(body, map[string]map[string]any{
			"$match": {
				"metadata[" + key + "]": value,
			},
		})
	}

	request := operations.V2ListAccountsRequest{
		Ledger: fctl.GetString(cmd, internal.LedgerFlag),
		RequestBody: map[string]any{
			"$and": body,
		},
	}
	rsp, err := stackClient.Ledger.V2.ListAccounts(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Accounts = rsp.V2AccountsCursorResponse.Cursor.Data

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {

	tableData := fctl.Map(c.store.Accounts, func(account shared.V2Account) []string {
		return []string{
			account.Address,
			fctl.MetadataAsShortString(account.Metadata),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"Address", "Metadata"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
