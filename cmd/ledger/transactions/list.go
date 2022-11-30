package transactions

import (
	"fmt"
	"strings"
	"time"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	internal "github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
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

	return internal2.NewCommand("list",
		internal2.WithAliases("ls", "l"),
		internal2.WithShortDescription("List transactions"),
		internal2.WithStringFlag(listTransactionAccountFlag, "", "Filter on account"),
		internal2.WithStringFlag(listTransactionDestinationFlag, "", "Filter on destination account"),
		internal2.WithStringFlag(listTransactionsAfterFlag, "", "Filter results after given tx id"),
		internal2.WithStringFlag(listTransactionsEndTimeFlag, "", "Consider transactions before date"),
		internal2.WithStringFlag(listTransactionsStartTimeFlag, "", "Consider transactions after date"),
		internal2.WithStringFlag(listTransactionSourceFlag, "", "Filter on source account"),
		internal2.WithStringFlag(listTransactionsReferenceFlag, "", "Filter on reference"),
		internal2.WithStringSliceFlag(listTransactionsMetadataFlag, []string{}, "Filter transactions with metadata"),
		internal2.WithIntFlag(listTransactionsPageSizeFlag, 5, "Page size"),
		// SDK not generating correct requests
		internal2.WithHiddenFlag(listTransactionsMetadataFlag),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			metadata := map[string]interface{}{}
			for _, v := range internal2.GetStringSlice(cmd, listTransactionsMetadataFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed metadata: %s", v)
				}
				metadata[parts[0]] = parts[1]
			}

			ledger := internal2.GetString(cmd, internal.LedgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.
				ListTransactions(cmd.Context(), ledger).
				PageSize(int32(internal2.GetInt(cmd, listTransactionsPageSizeFlag))).
				Reference(internal2.GetString(cmd, listTransactionsReferenceFlag)).
				Account(internal2.GetString(cmd, listTransactionAccountFlag)).
				Destination(internal2.GetString(cmd, listTransactionDestinationFlag)).
				Source(internal2.GetString(cmd, listTransactionSourceFlag)).
				After(internal2.GetString(cmd, listTransactionsAfterFlag)).
				EndTime(internal2.GetString(cmd, listTransactionsEndTimeFlag)).
				StartTime(internal2.GetString(cmd, listTransactionsStartTimeFlag)).
				Metadata(metadata).
				Execute()
			if err != nil {
				return err
			}

			tableData := internal2.Map(rsp.Cursor.Data, func(tx formance.Transaction) []string {
				return []string{
					fmt.Sprintf("%d", tx.Txid),
					func() string {
						if tx.Reference == nil {
							return ""
						}
						return *tx.Reference
					}(),
					tx.Timestamp.Format(time.RFC3339),
				}
			})
			tableData = internal2.Prepend(tableData, []string{"ID", "Reference", "Date"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
