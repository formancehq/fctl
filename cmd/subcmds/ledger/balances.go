package ledger

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerBalancesCommand() *cobra.Command {
	const (
		afterFlag   = "after"
		addressFlag = "address"
	)
	return cmdbuilder.NewCommand("balances",
		cmdbuilder.WithStringFlag(addressFlag, "", "Filter on specific address"),
		cmdbuilder.WithStringFlag(afterFlag, "", "Filter after specific address"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			balances, _, err := client.BalancesApi.
				GetBalances(cmd.Context(), viper.GetString(internal.LedgerFlag)).
				After(viper.GetString(afterFlag)).
				Address(viper.GetString(addressFlag)).
				Execute()
			if err != nil {
				return err
			}

			for _, accountBalances := range balances.Cursor.Data {
				for account, volumes := range accountBalances {
					fmt.Fprintln(cmd.OutOrStdout(), "Account:", account)
					for asset, balance := range volumes {
						fmt.Fprintf(cmd.OutOrStdout(), "\t%s: %d\n", asset, balance)
					}
				}
			}

			return nil
		}),
	)
}
