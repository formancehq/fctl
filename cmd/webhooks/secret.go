package webhooks

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewChangeSecretCommand() *cobra.Command {
	return internal.NewCommand("change-secret CONFIG_ID [SECRET]",
		internal.WithShortDescription("Change the signing secret of a config"),
		internal.WithAliases("cs"),
		internal.WithArgs(cobra.RangeArgs(1, 2)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			configID := args[0]
			secret := ""
			if len(args) > 1 {
				secret = args[1]
			}

			res, _, err := client.WebhooksApi.
				ChangeOneConfigSecret(cmd.Context(), configID).
				ChangeOneConfigSecretRequest(
					formance.ChangeOneConfigSecretRequest{
						Secret: secret,
					}).
				Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(),
				"Config updated successfully with new secret: %s", *res.Data.Secret)
			return nil
		}),
	)
}
