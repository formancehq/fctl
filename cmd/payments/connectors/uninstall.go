package connectors

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/openapi"
	"github.com/formancehq/fctl/cmd/payments/internal"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	return cmdbuilder.NewCommand("uninstall connector",
		cmdbuilder.WithAliases("uninstall", "u", "un"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithValidArgs("stripe"),
		cmdbuilder.WithShortDescription("Uninstall a connector"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal.NewPaymentsClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = client.DefaultApi.UninstallConnector(cmd.Context(), args[0]).Execute()
			if err != nil {
				return openapi.WrapError(err, "uninstalling connector")
			}
			cmdbuilder.Success(cmd.OutOrStdout(), "Connector '%s' uninstalled!", args[0])
			return nil
		}),
	)
}
