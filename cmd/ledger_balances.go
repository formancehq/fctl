package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerBalancesCommand() *cobra.Command {
	const (
		afterFlag   = "after"
		addressFlag = "address"
	)
	return newCommand("balances",
		withStringFlag(addressFlag, "", "Filter on specific address"),
		withStringFlag(afterFlag, "", "Filter after specific address"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			config, err := getConfig()
			if err != nil {
				return err
			}
			client, err := newLedgerClient(cmd, config)
			if err != nil {
				return err
			}

			balances, _, err := client.BalancesApi.
				GetBalances(cmd.Context(), viper.GetString(ledgerFlag)).
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
