package accounts

import (
	"github.com/formancehq/fctl/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewSetMetadataCommand() *cobra.Command {
	return fctl.NewCommand("set-metadata [ACCOUNT] [METADATA_KEY]=[METADATA_VALUE]...",
		fctl.WithShortDescription("Set metadata on account"),
		fctl.WithAliases("sm", "set-meta"),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {

			metadata, err := internal.ParseMetadata(args[1:])
			if err != nil {
				return err
			}

			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			account := args[0]

			_, err = ledgerClient.AccountsApi.
				AddMetadataToAccount(cmd.Context(), fctl.GetString(cmd, internal.LedgerFlag), account).
				RequestBody(metadata).
				Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(), "Metadata added!")
			return nil
		}),
	)
}
