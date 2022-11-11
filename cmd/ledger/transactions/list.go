package transactions

import (
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	internal2 "github.com/formancehq/fctl/cmd/ledger/internal"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithShortDescription("List transactions"),
		cmdbuilder.WithStringFlag(listTransactionAccountFlag, "", "Filter on account"),
		cmdbuilder.WithStringFlag(listTransactionDestinationFlag, "", "Filter on destination account"),
		cmdbuilder.WithStringFlag(listTransactionsAfterFlag, "", "Filter results after given tx id"),
		cmdbuilder.WithStringFlag(listTransactionsEndTimeFlag, "", "Consider transactions before date"),
		cmdbuilder.WithStringFlag(listTransactionsStartTimeFlag, "", "Consider transactions after date"),
		cmdbuilder.WithStringFlag(listTransactionSourceFlag, "", "Filter on source account"),
		cmdbuilder.WithStringFlag(listTransactionsReferenceFlag, "", "Filter on reference"),
		cmdbuilder.WithStringSliceFlag(listTransactionsMetadataFlag, []string{}, "Filter transactions with metadata"),
		cmdbuilder.WithIntFlag(listTransactionsPageSizeFlag, 5, "Page size"),
		// SDK not generating correct requests
		cmdbuilder.WithHiddenFlag(listTransactionsMetadataFlag),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewLedgerClient(cmd, cfg)
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

			ledger := viper.GetString(internal2.LedgerFlag)
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

			tableData := collections.Map(rsp.Cursor.Data, func(tx ledgerclient.Transaction) []string {
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
			tableData = collections.Prepend(tableData, []string{"ID", "Reference", "Date"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
