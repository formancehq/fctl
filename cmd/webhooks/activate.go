package webhooks

import (
	"net/url"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewActivateCommand() *cobra.Command {
	return fctl.NewCommand("activate",
		fctl.WithShortDescription("Activate one config"),
		fctl.WithAliases("ac", "a"),
		fctl.WithArgs(cobra.ExactArgs(1)),
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

			client, err := fctl.NewStackClient(cmd, cfg, stack)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			_, _, err = client.WebhooksApi.ActivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(), "Config activated successfully")
			return nil
		}),
	)
}
