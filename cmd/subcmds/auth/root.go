package auth

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/auth/clients"
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("auth",
		cmdbuilder.WithChildCommands(
			clients.NewAuthClientsCommand(),
		),
	)
}
