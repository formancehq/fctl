package cmd

import (
	"github.com/spf13/cobra"
)

func newAuthClientsSecretsCommand() *cobra.Command {
	return newCommand("secrets",
		withChildCommands(
			newAuthClientsSecretsCreateCommand(),
			newAuthClientsSecretsDeleteCommand(),
		),
	)
}
