package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("profiles",
		cmdbuilder.WithAliases("p", "prof"),
		cmdbuilder.WithShortDescription("Profiles management"),
		cmdbuilder.WithChildCommands(
			NewDeleteCommand(),
			NewListCommand(),
			NewRenameCommand(),
			NewShowCommand(),
			NewUseCommand(),
			NewSetDefaultOrganizationCommand(),
		),
	)
}
