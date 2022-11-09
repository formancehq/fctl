package cmd

import (
	"github.com/spf13/cobra"
)

func newSandboxCommand() *cobra.Command {
	return newMembershipCommand("sandbox",
		withShortDescription("manage your sandbox"),
		withAliases("stack"),
		withChildCommands(
			newSandboxCreateCommand(),
			newSandboxListCommand(),
			newSandboxDeleteCommand(),
			newSandboxShowCommand(),
		),
	)
}
