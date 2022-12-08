package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete [CONFIG_ID]",
		fctl.WithShortDescription("Delete a config"),
		fctl.WithAliases("del"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return errors.Wrap(err, "fctl.GetConfig")
			}

			webhookClient, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "fctl.NewStackClient")
			}

			_, err = webhookClient.WebhooksApi.DeleteOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return errors.Wrap(err, "deleting config")
			}

			fctl.Success(cmd.OutOrStdout(), "Config deleted successfully")
			return nil
		}),
	)
}
