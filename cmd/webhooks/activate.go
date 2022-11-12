package webhooks

import (
	"net/url"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewActivateCommand() *cobra.Command {
	return cmdbuilder.NewCommand("activate",
		cmdbuilder.WithShortDescription("Activate a webhook"),
		cmdbuilder.WithAliases("ac", "a"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
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

			_, _, err = webhookClient.ConfigsApi.ActivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Config activated")
			return nil
		}),
	)
}
