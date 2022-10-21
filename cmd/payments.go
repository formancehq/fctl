package cmd

import (
	"github.com/spf13/cobra"
)

func newPaymentsCommand() *cobra.Command {
	return newStackCommand("payments",
		withChildCommands(
			newPaymentsConnectorsCommand(),
		),
	)
}
