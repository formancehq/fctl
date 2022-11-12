package transactions

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/spf13/cobra"
)

func NewSetMetadataCommand() *cobra.Command {
	return cmdbuilder.NewCommand("set-metadata [TRANSACTION] [METADATA_KEY]=[METADATA_VALUE]...",
		cmdbuilder.WithShortDescription("Set metadata on transaction"),
		cmdbuilder.WithAliases("sm", "set-meta"),
		cmdbuilder.WithValidArgs("last"),
		cmdbuilder.WithArgs(cobra.MinimumNArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			metadata, err := internal.ParseMetadata(args[1:])
			if err != nil {
				return err
			}

			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			transactionID, err := internal.TransactionIDOrLastN(cmd.Context(), ledgerClient,
				cmdutils.Viper(cmd.Context()).GetString(internal.LedgerFlag), args[0])
			if err != nil {
				return err
			}

			_, err = ledgerClient.TransactionsApi.
				AddMetadataOnTransaction(cmd.Context(), cmdutils.Viper(cmd.Context()).GetString(internal.LedgerFlag), int32(transactionID)).
				RequestBody(metadata).
				Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Metadata added!")
			return nil
		}),
	)
}
