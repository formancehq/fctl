package cmd

import (
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	return newStackCommand("auth",
		withChildCommands(
			newAuthClientsCommand(),
		),
	)
}
