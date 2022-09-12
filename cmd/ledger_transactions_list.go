package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

var listTransactionsCommand = &cobra.Command{
	Use:   "list",
	Short: "list transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
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
			printTransaction(cmd, s)
		}
		return nil
	},
}

func init() {
	listTransactionsCommand.Flags().Int(listTransactionsPageSizeFlag, 15, "Page size")
	listTransactionsCommand.Flags().String(listTransactionAccountFlag, "", "Filter on account")
	listTransactionsCommand.Flags().String(listTransactionDestinationFlag, "", "Filter on destination account")
	listTransactionsCommand.Flags().StringSlice(listTransactionsMetadataFlag, []string{}, "Filter transactions with metadata")
	listTransactionsCommand.Flags().String(listTransactionsAfterFlag, "", "Filter results after given tx id")
	listTransactionsCommand.Flags().String(listTransactionsEndTimeFlag, "", "Consider transactions before date")
	listTransactionsCommand.Flags().String(listTransactionsStartTimeFlag, "", "Consider transactions after date")
	listTransactionsCommand.Flags().String(listTransactionSourceFlag, "", "Filter on source account")
	listTransactionsCommand.Flags().String(listTransactionsReferenceFlag, "", "Filter on reference")

	// SDK not generating correct requests
	listTransactionsCommand.Flags().MarkHidden(listTransactionsMetadataFlag)

	transactionsCommand.AddCommand(listTransactionsCommand)
}
