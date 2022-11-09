package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerTransactionsListCommand() *cobra.Command {
	const (
		listTransactionsPageSizeFlag   = "page-size"
		listTransactionsMetadataFlag   = "metadata"
		listTransactionsReferenceFlag  = "reference"
		listTransactionAccountFlag     = "account"
		listTransactionDestinationFlag = "dst"
		listTransactionSourceFlag      = "src"
		listTransactionsAfterFlag      = "after"
		listTransactionsEndTimeFlag    = "end"
		listTransactionsStartTimeFlag  = "start"
	)

	return newCommand("list",
		withShortDescription("list transactions"),
		withStringFlag(listTransactionAccountFlag, "", "Filter on account"),
		withStringFlag(listTransactionDestinationFlag, "", "Filter on destination account"),
		withStringFlag(listTransactionsAfterFlag, "", "Filter results after given tx id"),
		withStringFlag(listTransactionsEndTimeFlag, "", "Consider transactions before date"),
		withStringFlag(listTransactionsStartTimeFlag, "", "Consider transactions after date"),
		withStringFlag(listTransactionSourceFlag, "", "Filter on source account"),
		withStringFlag(listTransactionsReferenceFlag, "", "Filter on reference"),
		withStringSliceFlag(listTransactionsMetadataFlag, []string{}, "Filter transactions with metadata"),
		withIntFlag(listTransactionsPageSizeFlag, 15, "Page size"),
		// SDK not generating correct requests
		withHiddenFlag(listTransactionsMetadataFlag),
		withRunE(func(cmd *cobra.Command, args []string) error {
			ledgerClient, err := newLedgerClient(cmd)
			if err != nil {
				return err
			}

			metadata, err := metadataFromFlag(listTransactionsMetadataFlag)
			if err != nil {
				return err
			}

			ledger := viper.GetString(ledgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.
				ListTransactions(cmd.Context(), ledger).
				PageSize(int32(viper.GetInt(listTransactionsPageSizeFlag))).
				Reference(viper.GetString(listTransactionsReferenceFlag)).
				Account(viper.GetString(listTransactionAccountFlag)).
				Destination(viper.GetString(listTransactionDestinationFlag)).
				Source(viper.GetString(listTransactionSourceFlag)).
				After(viper.GetString(listTransactionsAfterFlag)).
				EndTime(viper.GetString(listTransactionsEndTimeFlag)).
				StartTime(viper.GetString(listTransactionsStartTimeFlag)).
				Metadata(metadata).
				Execute()
			if err != nil {
				return err
			}
			if len(rsp.Cursor.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No transactions found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Transactions: ")
			for _, s := range rsp.Cursor.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> Transaction: %d\r\n", s.Txid)
				internal.PrintLedgerTransaction(cmd.OutOrStdout(), s)
			}
			return nil
		}),
	)
}
