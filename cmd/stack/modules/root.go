package modules

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
