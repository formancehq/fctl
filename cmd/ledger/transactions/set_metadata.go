package transactions

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/spf13/cobra"
)

func NewSetMetadataCommand() *cobra.Command {
	return internal2.NewCommand("set-metadata [TRANSACTION] [METADATA_KEY]=[METADATA_VALUE]...",
		internal2.WithShortDescription("Set metadata on transaction"),
		internal2.WithAliases("sm", "set-meta"),
		internal2.WithValidArgs("last"),
		internal2.WithArgs(cobra.MinimumNArgs(2)),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {

			metadata, err := internal.ParseMetadata(args[1:])
			if err != nil {
				return err
			}

			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			transactionID, err := internal.TransactionIDOrLastN(cmd.Context(), ledgerClient,
				internal2.GetString(cmd, internal.LedgerFlag), args[0])
			if err != nil {
				return err
			}

			_, err = ledgerClient.TransactionsApi.
				AddMetadataOnTransaction(cmd.Context(), internal2.GetString(cmd, internal.LedgerFlag), int32(transactionID)).
				RequestBody(metadata).
				Execute()
			if err != nil {
				return err
			}

			internal2.Success(cmd.OutOrStdout(), "Metadata added!")
			return nil
		}),
	)
}
