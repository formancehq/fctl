package cmd

import (
	"github.com/spf13/cobra"
)

func newAuthClientsCommand() *cobra.Command {
	return newCommand("clients",
		withChildCommands(
			newAuthClientsListCommand(),
			newAuthClientsCreateCommand(),
			newAuthClientsDeleteCommand(),
			newAuthClientsUpdateCommand(),
			newAuthClientsSecretsCommand(),
			newAuthClientsShowCommand(),
		),
	)
}
