package webhooks

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/webhooks/client"
	"github.com/spf13/cobra"
)

func NewChangeSecretCommand() *cobra.Command {
	return cmdbuilder.NewCommand("change-secret CONFIG_ID [SECRET]",
		cmdbuilder.WithShortDescription("Change the secret of a webhook"),
		cmdbuilder.WithAliases("cs"),
		cmdbuilder.WithArgs(cobra.RangeArgs(1, 2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			webhookClient, err := newWebhookClient(cmd, cfg)
			if err != nil {
				return err
			}

			configID := args[0]
			secret := ""
			if len(args) > 1 {
				secret = args[1]
			}

			response, _, err := webhookClient.ConfigsApi.
				ChangeOneConfigSecret(cmd.Context(), configID).
				ChangeOneConfigSecretRequest(client.ChangeOneConfigSecretRequest{
					Secret: &secret,
				}).
				Execute()
			if err != nil {
				return err
			}

			newSecret := *response.Cursor.Data[0].Secret

			cmdbuilder.Success(cmd.OutOrStdout(), "Config updated with secret: %s", newSecret)
			return nil
		}),
	)
}
