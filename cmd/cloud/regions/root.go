package regions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("regions",
		fctl.WithAliases("region", "reg"),
		fctl.WithShortDescription("Regions management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
		),
	)
}
