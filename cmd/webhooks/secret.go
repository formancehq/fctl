package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewChangeSecretCommand() *cobra.Command {
	return fctl.NewCommand("change-secret CONFIG_ID [SECRET]",
		fctl.WithShortDescription("Change the signing secret of a config"),
		fctl.WithAliases("cs"),
		fctl.WithArgs(cobra.RangeArgs(1, 2)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.Get(cmd)
			if err != nil {
				return err
			}

			client, err := fctl.NewStackClient(cmd, cfg)
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

			fctl.Success(cmd.OutOrStdout(),
				"Config updated successfully with new secret: %s", *res.Data.Secret)
			return nil
		}),
	)
}
