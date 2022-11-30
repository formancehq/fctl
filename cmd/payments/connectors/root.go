package connectors

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/payments/connectors/install"
	"github.com/spf13/cobra"
)

func NewConnectorsCommand() *cobra.Command {
	return internal.NewCommand("connectors",
		internal.WithAliases("c", "co", "con"),
		internal.WithShortDescription("Connectors management"),
		internal.WithChildCommands(
			NewGetConfigCommand(),
			NewUninstallCommand(),
			install.NewInstallCommand(),
		),
	)
}
