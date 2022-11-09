package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerAccountsListCommand() *cobra.Command {
	return newCommand("list",
		withShortDescription("list accounts"),
		withRunE(func(cmd *cobra.Command, args []string) error {

			ledgerClient, err := newLedgerClient(cmd)
			if err != nil {
				return err
			}

			ledger := viper.GetString(ledgerFlag)
			rsp, _, err := ledgerClient.AccountsApi.ListAccounts(cmd.Context(), ledger).Execute()
			if err != nil {
				return err
			}
			if len(rsp.Cursor.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No accounts found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Accounts: ")
			for _, account := range rsp.Cursor.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> Account: %s\r\n", account.Address)
				if account.Metadata != nil && len(*account.Metadata) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
					for k, v := range *account.Metadata {
						fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
					}
				}
			}
			return nil
		}),
	)
}
