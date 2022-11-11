package users

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("users",
		cmdbuilder.WithAliases("u"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
