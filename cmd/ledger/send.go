package ledger

import (
	"strconv"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewSendCommand() *cobra.Command {
	const (
		metadataFlag  = "metadata"
		referenceFlag = "reference"
	)
	return internal2.NewCommand("send [SOURCE] [DESTINATION] [AMOUNT] [ASSET]",
		internal2.WithAliases("s", "se"),
		internal2.WithShortDescription("Send from one account to another"),
		internal2.WithArgs(cobra.RangeArgs(3, 4)),
		internal2.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		internal2.WithStringFlag(referenceFlag, "", "Reference to add to the generated transaction"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
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

			metadata, err := internal.ParseMetadata(internal2.GetStringSlice(cmd, metadataFlag))
			if err != nil {
				return err
			}

			reference := internal2.GetString(cmd, referenceFlag)
			response, _, err := ledgerClient.TransactionsApi.
				CreateTransaction(cmd.Context(), internal2.GetString(cmd, internal.LedgerFlag)).
				PostTransaction(formance.PostTransaction{
					Postings: []formance.Posting{{
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
