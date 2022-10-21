package cmd

import (
	"github.com/spf13/cobra"
)

func newSandboxCommand() *cobra.Command {
	return newMembershipCommand("sandbox",
		withShortDescription("manage your sandbox"),
		withChildCommands(
			newSandboxCreateCommand(),
			newSandboxListCommand(),
			newSandboxDeleteCommand(),
		),
	)
}
