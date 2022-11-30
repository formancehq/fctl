package users

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("users",
		internal.WithShortDescription("Users management"),
		internal.WithAliases("u", "user"),
		internal.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
