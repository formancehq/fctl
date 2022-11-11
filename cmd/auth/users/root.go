package users

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("users",
		cmdbuilder.WithShortDescription("Users management"),
		cmdbuilder.WithAliases("u"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
