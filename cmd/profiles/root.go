package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("profiles",
		internal.WithAliases("p", "prof"),
		internal.WithShortDescription("Profiles management"),
		internal.WithChildCommands(
			NewDeleteCommand(),
			NewListCommand(),
			NewRenameCommand(),
			NewShowCommand(),
			NewUseCommand(),
			NewSetDefaultOrganizationCommand(),
		),
	)
}
