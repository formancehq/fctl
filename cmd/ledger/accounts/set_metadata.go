package accounts

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewSetMetadataCommand() *cobra.Command {
	return cmdbuilder.NewCommand("set-metadata [ACCOUNT] [METADATA_KEY]=[METADATA_VALUE]...",
		cmdbuilder.WithShortDescription("Set metadata on account"),
		cmdbuilder.WithAliases("sm", "set-meta"),
		cmdbuilder.WithArgs(cobra.MinimumNArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			metadata, err := internal.ParseMetadata(args[1:])
			if err != nil {
				return err
			}

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			account := args[0]

			_, err = ledgerClient.AccountsApi.
				AddMetadataToAccount(cmd.Context(), viper.GetString(internal.LedgerFlag), account).
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
