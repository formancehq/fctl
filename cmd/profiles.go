package cmd

import (
	"github.com/spf13/cobra"
)

func newProfilesCommand() *cobra.Command {
	return newCommand("profiles",
		withAliases("p"),
		withChildCommands(
			newProfilesDeleteCommand(),
			newProfilesListCommand(),
			newProfilesRenameCommand(),
			newProfilesShowCommand(),
			newProfilesUseCommand(),
		),
	)
}
