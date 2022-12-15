package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewChangeSecretCommand() *cobra.Command {
	return fctl.NewCommand("change-secret CONFIG_ID [SECRET]",
		fctl.WithShortDescription("Change the signing secret of a config"),
		fctl.WithConfirmFlag(),
		fctl.WithAliases("cs"),
		fctl.WithArgs(cobra.RangeArgs(1, 2)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			organizationID, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := fctl.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			if !fctl.CheckStackApprobation(cmd, stack, "You are about to change a webhook secret") {
				return fctl.ErrMissingApproval
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
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
