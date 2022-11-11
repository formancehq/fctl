package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("profiles",
		cmdbuilder.WithAliases("p"),
		cmdbuilder.WithChildCommands(
			newProfilesDeleteCommand(),
			newProfilesListCommand(),
			newProfilesRenameCommand(),
			newProfilesShowCommand(),
			newProfilesUseCommand(),
		),
	)
}
