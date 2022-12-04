package connectors

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	return fctl.NewCommand("uninstall connector",
		fctl.WithAliases("uninstall", "u", "un"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgs("stripe"),
		fctl.WithShortDescription("Uninstall a connector"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := fctl.Get(cmd)
			if err != nil {
				return err
			}

			client, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = client.PaymentsApi.UninstallConnector(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fctl.WrapError(err, "uninstalling connector")
			}
			fctl.Success(cmd.OutOrStdout(), "Connector '%s' uninstalled!", args[0])
			return nil
		}),
	)
}
