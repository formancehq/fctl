package applications

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
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
