package webhooks

import (
	"net/url"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete",
		fctl.WithShortDescription("Delete a config"),
		fctl.WithAliases("del"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.Get(cmd)
			if err != nil {
				return err
			}

			webhookClient, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			_, err = webhookClient.WebhooksApi.DeleteOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(), "Config deleted successfully")
			return nil
		}),
	)
}
