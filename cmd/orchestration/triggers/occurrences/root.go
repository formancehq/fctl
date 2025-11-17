package occurrences

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("occurrences",
		fctl.WithAliases("oc", "o"),
		fctl.WithShortDescription("Triggers occurrences management"),
		fctl.WithChildCommands(
			NewListCommand(),
		),
	)
}
