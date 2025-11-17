package workflows

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("workflows",
		fctl.WithAliases("w", "work"),
		fctl.WithShortDescription("Workflows management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewRunCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
		),
	)
}
