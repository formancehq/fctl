package cmd

import (
	"fmt"
	"strconv"
	"time"

	ledgerclient "github.com/numary/ledger/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func printTransaction(cmd *cobra.Command, tx ledgerclient.Transaction) {
	fmt.Fprintf(cmd.OutOrStdout(), "Date: %s\r\n", tx.Timestamp.Format(time.RFC3339))
	if tx.Reference != nil && *tx.Reference != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Reference: %s", *tx.Reference)
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
	Use:   "transactions",
	Short: "Manage transactions (create/read)",
}

var revertTransactionCommand = &cobra.Command{
	Use:   "revert [TXID]",
	Short: "revert a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		txId, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return errors.Wrapf(err, "parsing txid")
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.TransactionsApi.RevertTransaction(cmd.Context(), ledger, int32(txId)).Execute()
		if err != nil {
			return errors.Wrapf(err, "reverting transaction")
		}
		printTransaction(cmd, rsp.Data)
		return nil
	},
}

var showTransactionCommand = &cobra.Command{
	Use:   "show [TXID]",
	Short: "print a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		txId, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return errors.Wrapf(err, "parsing txid")
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
		if err != nil {
			return errors.Wrapf(err, "retrieving transaction")
		}
		printTransaction(cmd, rsp.Data)
		return nil
	},
}

func init() {
	transactionsCommand.PersistentFlags().String(stackFlag, "", "Specific stack (not required if only one stack is present)")
	transactionsCommand.PersistentFlags().String(ledgerFlag, "default", "Specific ledger ")

	transactionsCommand.AddCommand(revertTransactionCommand, showTransactionCommand)
	rootCommand.AddCommand(transactionsCommand)
}
