package transactions

import (
	"fmt"
	"strings"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerTransactionsListCommand() *cobra.Command {
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

	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithShortDescription("list transactions"),
		cmdbuilder.WithStringFlag(listTransactionAccountFlag, "", "Filter on account"),
		cmdbuilder.WithStringFlag(listTransactionDestinationFlag, "", "Filter on destination account"),
		cmdbuilder.WithStringFlag(listTransactionsAfterFlag, "", "Filter results after given tx id"),
		cmdbuilder.WithStringFlag(listTransactionsEndTimeFlag, "", "Consider transactions before date"),
		cmdbuilder.WithStringFlag(listTransactionsStartTimeFlag, "", "Consider transactions after date"),
		cmdbuilder.WithStringFlag(listTransactionSourceFlag, "", "Filter on source account"),
		cmdbuilder.WithStringFlag(listTransactionsReferenceFlag, "", "Filter on reference"),
		cmdbuilder.WithStringSliceFlag(listTransactionsMetadataFlag, []string{}, "Filter transactions with metadata"),
		cmdbuilder.WithIntFlag(listTransactionsPageSizeFlag, 15, "Page size"),
		// SDK not generating correct requests
		cmdbuilder.WithHiddenFlag(listTransactionsMetadataFlag),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			metadata := map[string]interface{}{}
			for _, v := range viper.GetStringSlice(listTransactionsMetadataFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed metadata: %s", v)
				}
				metadata[parts[0]] = parts[1]
			}

			ledger := viper.GetString(internal.LedgerFlag)
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
