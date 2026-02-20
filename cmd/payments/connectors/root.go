package connectors

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/configs"
	"github.com/formancehq/fctl/v3/cmd/payments/connectors/install"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewConnectorsCommand() *cobra.Command {
	return fctl.NewCommand("connectors",
		fctl.WithAliases("c", "co", "con"),
		fctl.WithShortDescription("Connectors management"),
		fctl.WithChildCommands(
			NewUninstallCommand(),
			NewListCommand(),
			install.NewInstallCommand(),
			configs.NewUpdateConfigCommands(),
			configs.NewLoadConfigCommand(),
		),
	)
}
