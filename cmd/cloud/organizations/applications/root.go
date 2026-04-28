package applications

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("applications",
		fctl.WithAliases("apps", "app"),
		fctl.WithShortDescription("Applications management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
