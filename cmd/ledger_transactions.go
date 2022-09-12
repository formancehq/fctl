package cmd

import (
	"fmt"
	"time"

	ledgerclient "github.com/numary/ledger/client"
	"github.com/spf13/cobra"
)

func printTransaction(cmd *cobra.Command, tx ledgerclient.Transaction) {
	fmt.Fprintf(cmd.OutOrStdout(), "Date: %s\r\n", tx.Timestamp.Format(time.RFC3339))
	if tx.Reference != nil && *tx.Reference != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Reference: %s\r\n", *tx.Reference)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Pre commit volumes:")
	for account, v := range *tx.PreCommitVolumes {
		fmt.Fprintf(cmd.OutOrStdout(), "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Post commit volumes:")
	for account, v := range *tx.PostCommitVolumes {
		fmt.Fprintf(cmd.OutOrStdout(), "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	if len(tx.Metadata) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
		for k, v := range tx.Metadata {
			fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
		}
	}
}

var transactionsCommand = &cobra.Command{
	Use: "transactions",
}

func init() {
	ledgerCommand.AddCommand(transactionsCommand)
}
