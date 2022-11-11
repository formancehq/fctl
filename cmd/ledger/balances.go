package ledger

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	internal2 "github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerBalancesCommand() *cobra.Command {
	const (
		afterFlag   = "after"
		addressFlag = "address"
	)
	return cmdbuilder.NewCommand("balances",
		cmdbuilder.WithAliases("balance", "bal", "b"),
		cmdbuilder.WithStringFlag(addressFlag, "", "Filter on specific address"),
		cmdbuilder.WithStringFlag(afterFlag, "", "Filter after specific address"),
		cmdbuilder.WithShortDescription("Read balances"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := internal2.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			balances, _, err := client.BalancesApi.
				GetBalances(cmd.Context(), viper.GetString(internal2.LedgerFlag)).
				After(viper.GetString(afterFlag)).
				Address(viper.GetString(addressFlag)).
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
