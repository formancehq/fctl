package connectors

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/payments/connectors/install"
	"github.com/spf13/cobra"
)

func NewConnectorsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("connectors",
		cmdbuilder.WithAliases("c", "co", "con"),
		cmdbuilder.WithShortDescription("Connectors management"),
		cmdbuilder.WithChildCommands(
			NewGetConfigCommand(),
			NewUninstallCommand(),
			install.NewInstallCommand(),
		),
	)
}
