package plugin

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("plugin",
		fctl.WithShortDescription("Manage fctl plugins"),
		fctl.WithChildCommands(
			NewInstallCommand(),
			NewListCommand(),
			NewUpdateCommand(),
			NewRemoveCommand(),
		),
	)
}
