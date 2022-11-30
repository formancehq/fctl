package stack

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewMembershipCommand("sandbox",
		internal.WithShortDescription("Manage your sandbox"),
		internal.WithAliases("stack", "stacks", "st"),
		internal.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewDeleteCommand(),
			NewShowCommand(),
		),
	)
}
