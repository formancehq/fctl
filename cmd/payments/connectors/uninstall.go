package connectors

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	return internal2.NewCommand("uninstall connector",
		internal2.WithAliases("uninstall", "u", "un"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithValidArgs("stripe"),
		internal2.WithShortDescription("Uninstall a connector"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = client.PaymentsApi.UninstallConnector(cmd.Context(), args[0]).Execute()
			if err != nil {
				return internal2.WrapError(err, "uninstalling connector")
			}
			internal2.Success(cmd.OutOrStdout(), "Connector '%s' uninstalled!", args[0])
			return nil
		}),
	)
}
