package cmd

import (
	"github.com/spf13/cobra"
)

func newPaymentsConnectorsInstallCommand() *cobra.Command {
	return newCommand("install",
		withChildCommands(
			newPaymentsConnectorsInstallStripeCommand(),
		),
	)
}
