package stack

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewMembershipCommand("sandbox",
		cmdbuilder.WithShortDescription("manage your sandbox"),
		cmdbuilder.WithAliases("stack"),
		cmdbuilder.WithChildCommands(
			newSandboxCreateCommand(),
			newSandboxListCommand(),
			newSandboxDeleteCommand(),
			newSandboxShowCommand(),
		),
	)
}
