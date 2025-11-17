package invitations

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("invitations",
		fctl.WithShortDescription("Invitations management"),
		fctl.WithAliases("invit", "inv", "i"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewAcceptCommand(),
			NewDeclineCommand(),
		),
	)
}
