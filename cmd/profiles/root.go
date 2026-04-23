package profiles

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("profiles",
		fctl.WithAliases("p", "prof", "profile"),
		fctl.WithShortDescription("Manage profiles"),
		fctl.WithChildCommands(
			NewDeleteCommand(),
			NewListCommand(),
			NewRenameCommand(),
			NewShowCommand(),
			NewUseCommand(),
			NewSetDefaultOrganizationCommand(),
			NewSetDefaultStackCommand(),
			NewResetCommand(),
		),
	)
}
