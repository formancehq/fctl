package stack

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewSandboxCommand() *cobra.Command {
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
