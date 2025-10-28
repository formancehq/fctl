package modules

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("modules",
		fctl.WithShortDescription("Manage your modules"),
		fctl.WithAliases("module", "mod"),
		fctl.WithChildCommands(
			NewDisableCommand(),
			NewEnableCommand(),
			NewListCommand(),
		),
	)
}
