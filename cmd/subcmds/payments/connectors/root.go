package connectors

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/payments/connectors/install"
	"github.com/spf13/cobra"
)

func NewPaymentsConnectorsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("connectors",
		cmdbuilder.WithShortDescription("Handle payments service connectors"),
		cmdbuilder.WithChildCommands(
			NewPaymentsConnectorsGetConfigCommand(),
			install.NewPaymentsConnectorsInstallCommand(),
		),
	)
}
