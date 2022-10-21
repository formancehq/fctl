package cmd

import (
	"github.com/spf13/cobra"
)

func newPaymentsConnectorsCommand() *cobra.Command {
	return newCommand("connectors",
		withShortDescription("Handle payments service connectors"),
		withChildCommands(
			newPaymentsConnectorsGetConfigCommand(),
			newPaymentsConnectorsInstallCommand(),
		),
	)
}
