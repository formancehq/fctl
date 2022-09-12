package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	balancesAfterFlag   = "after"
	balancesAddressFlag = "address"
)

var balancesCommand = &cobra.Command{
	Use: "balances",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		balances, _, err := client.BalancesApi.
			GetBalances(cmd.Context(), viper.GetString(ledgerFlag)).
			After(viper.GetString(balancesAfterFlag)).
			Address(viper.GetString(balancesAddressFlag)).
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
	},
}

func init() {
	balancesCommand.Flags().String(balancesAddressFlag, "", "Filter on specific address")
	balancesCommand.Flags().String(balancesAfterFlag, "", "Filter after specific address")
	ledgerCommand.AddCommand(balancesCommand)
}
