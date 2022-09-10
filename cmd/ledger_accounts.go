package cmd

import (
	"fmt"

	ledgerclient "github.com/numary/numary-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func printAccount(cmd *cobra.Command, account ledgerclient.AccountWithVolumesAndBalances) {
	if account.Volumes != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "Volumes:")
		for asset, v := range *account.Volumes {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %d\tOutput: %d\tBalance: %d\r\n",
				asset, v["input"], v["output"], v["balance"])
		}
	}

	if account.Metadata != nil && len(*account.Metadata) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
		for k, v := range *account.Metadata {
			fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
		}
	}
}

var accountsCommand = &cobra.Command{
	Use:   "accounts",
	Short: "handle ledger accounts",
}

var listAccountsCommand = &cobra.Command{
	Use:   "list",
	Short: "list accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
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
			fmt.Fprintf(cmd.OutOrStdout(), "-> Account: %account\r\n", account.Address)
			if account.Metadata != nil && len(*account.Metadata) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
				for k, v := range *account.Metadata {
					fmt.Fprintf(cmd.OutOrStdout(), "\t- %account: %account\r\n", k, v)
				}
			}
		}
		return nil
	},
}

var showAccountCommand = &cobra.Command{
	Use:   "show [ADDRESS]",
	Args:  cobra.ExactArgs(1),
	Short: "display account",
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.AccountsApi.GetAccount(cmd.Context(), ledger, args[0]).Execute()
		if err != nil {
			return err
		}

		printAccount(cmd, rsp.Data)
		return nil
	},
}

func init() {
	//TODO: Factorize
	accountsCommand.PersistentFlags().String(stackFlag, "", "Specific stack (not required if only one stack is present)")
	accountsCommand.PersistentFlags().String(ledgerFlag, "default", "Specific ledger ")
	accountsCommand.AddCommand(listAccountsCommand, showAccountCommand)
	rootCommand.AddCommand(accountsCommand)
}
