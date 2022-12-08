package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewActivateCommand() *cobra.Command {
	return fctl.NewCommand("activate [CONFIG_ID]",
		fctl.WithShortDescription("Activate one config"),
		fctl.WithAliases("ac", "a"),
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

			_, _, err = client.WebhooksApi.ActivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return errors.Wrap(err, "activating config")
			}

			fctl.Success(cmd.OutOrStdout(), "Config activated successfully")
			return nil
		}),
	)
}
