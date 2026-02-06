package invitations

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("invitations",
		fctl.WithAliases("invit", "inv", "i", "invitation"),
		fctl.WithShortDescription("Invitations management"),
		fctl.WithChildCommands(
			NewSendCommand(),
			NewListCommand(),
			NewDeleteCommand(),
		),
	)
}
