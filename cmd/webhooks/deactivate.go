package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeactivateCommand() *cobra.Command {
	return fctl.NewCommand("deactivate [CONFIG_ID]",
		fctl.WithShortDescription("Deactivate one config"),
		fctl.WithAliases("deac"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return errors.Wrap(err, "fctl.GetConfig")
			}

			client, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "fctl.NewStackClient")
			}

			_, _, err = client.WebhooksApi.DeactivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return errors.Wrap(err, "deactivating config")
			}

			fctl.Success(cmd.OutOrStdout(), "Config deactivated successfully")
			return nil
		}),
	)
}
