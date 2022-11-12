package webhooks

import (
	"net/url"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return cmdbuilder.NewCommand("delete",
		cmdbuilder.WithShortDescription("Delete a config"),
		cmdbuilder.WithAliases("del"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			webhookClient, err := newWebhookClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			_, err = webhookClient.ConfigsApi.DeleteOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Config deleted")
			return nil
		}),
	)
}
