package auth

import (
	"github.com/formancehq/fctl/cmd/auth/clients"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("auth",
		cmdbuilder.WithChildCommands(
			clients.NewAuthClientsCommand(),
		),
	)
}
