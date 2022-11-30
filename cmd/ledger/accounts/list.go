package accounts

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	internal "github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal2.NewCommand("list",
		internal2.WithAliases("ls", "l"),
		internal2.WithShortDescription("List accounts"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			ledger := internal2.GetString(cmd, internal.LedgerFlag)
			rsp, _, err := ledgerClient.AccountsApi.ListAccounts(cmd.Context(), ledger).Execute()
			if err != nil {
				return err
			}

			tableData := internal2.Map(rsp.Cursor.Data, func(account formance.Account) []string {
				return []string{
					account.Address,
				}
			})
			tableData = internal2.Prepend(tableData, []string{"Address"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
