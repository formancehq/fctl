package webhooks

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/webhooks/client"
	"github.com/spf13/cobra"
)

func NewChangeSecretCommand() *cobra.Command {
	return internal.NewCommand("change-secret CONFIG_ID [SECRET]",
		internal.WithShortDescription("Change the secret of a webhook"),
		internal.WithAliases("cs"),
		internal.WithArgs(cobra.RangeArgs(1, 2)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			webhookClient, err := NewStackClient(cmd, cfg)
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

			internal.Success(cmd.OutOrStdout(), "Config updated with secret: %s", newSecret)
			return nil
		}),
	)
}
