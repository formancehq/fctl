package users

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("users",
		internal.WithAliases("u"),
		internal.WithShortDescription("Users management"),
		internal.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
