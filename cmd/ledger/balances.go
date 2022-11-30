package ledger

import (
	"fmt"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	internal "github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewBalancesCommand() *cobra.Command {
	const (
		afterFlag   = "after"
		addressFlag = "address"
	)
	return internal2.NewCommand("balances",
		internal2.WithAliases("balance", "bal", "b"),
		internal2.WithStringFlag(addressFlag, "", "Filter on specific address"),
		internal2.WithStringFlag(afterFlag, "", "Filter after specific address"),
		internal2.WithShortDescription("Read balances"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			balances, _, err := client.BalancesApi.
				GetBalances(cmd.Context(), internal2.GetString(cmd, internal.LedgerFlag)).
				After(internal2.GetString(cmd, afterFlag)).
				Address(internal2.GetString(cmd, addressFlag)).
				Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{"Account", "Asset", "Balance"})
			for _, accountBalances := range balances.Cursor.Data {
				for account, volumes := range accountBalances {
					for asset, balance := range volumes {
						tableData = append(tableData, []string{
							account, asset, fmt.Sprint(balance),
						})
					}
				}
			}
			if err := pterm.DefaultTable.
				WithHasHeader(true).
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render(); err != nil {
				return err
			}

			return nil
		}),
	)
}
