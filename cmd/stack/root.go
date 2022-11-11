package stack

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewMembershipCommand("sandbox",
		cmdbuilder.WithShortDescription("Manage your sandbox"),
		cmdbuilder.WithAliases("stack", "stacks", "s", "st"),
		cmdbuilder.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewDeleteCommand(),
			NewShowCommand(),
		),
	)
}
