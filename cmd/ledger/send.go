package ledger

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/numary/ledger/client"
	"github.com/spf13/cobra"
)

func NewSendCommand() *cobra.Command {
	const (
		metadataFlag  = "metadata"
		referenceFlag = "reference"
	)
	return cmdbuilder.NewCommand("send [SOURCE] [DESTINATION] [AMOUNT] [ASSET]",
		cmdbuilder.WithAliases("s", "se"),
		cmdbuilder.WithShortDescription("Send from one account to another"),
		cmdbuilder.WithArgs(cobra.RangeArgs(3, 4)),
		cmdbuilder.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		cmdbuilder.WithStringFlag(referenceFlag, "", "Reference to add to the generated transaction"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			var source, destination, asset, amountStr string
			if len(args) == 3 {
				source = "world"
				destination = args[0]
				amountStr = args[1]
				asset = args[2]
			} else {
				source = args[0]
				destination = args[1]
				amountStr = args[2]
				asset = args[3]
			}

			amount, err := strconv.ParseInt(amountStr, 10, 32)
			if err != nil {
				return err
			}

			metadata, err := internal.ParseMetadata(cmdutils.Viper(cmd.Context()).GetStringSlice(metadataFlag))
			if err != nil {
				return err
			}

			reference := cmdutils.Viper(cmd.Context()).GetString(referenceFlag)
			response, _, err := ledgerClient.TransactionsApi.
				CreateTransaction(cmd.Context(), cmdutils.Viper(cmd.Context()).GetString(internal.LedgerFlag)).
				TransactionData(client.TransactionData{
					Postings: []client.Posting{{
						Amount:      int32(amount),
						Asset:       asset,
						Destination: destination,
						Source:      source,
					}},
					Reference: &reference,
					Metadata:  metadata,
				}).
				Execute()
			if err != nil {
				return err
			}

			return internal.PrintTransaction(cmd.OutOrStdout(), response.Data[0])
		}),
	)
}
