package invitations

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewOrganizationsInvitationsCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("invitations",
		cmdbuilder.WithAliases("invit", "inv", "i"),
		cmdbuilder.WithShortDescription("Invitations management"),
		cmdbuilder.WithChildCommands(
			NewOrganizationsInvitationsSendCommand(),
			NewOrganizationsInvitationsListCommand(),
		),
	)
}
