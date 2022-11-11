package cmd

import (
	"github.com/spf13/cobra"
)

func newOrganizationsInvitationsCommand() *cobra.Command {
	return newStackCommand("invitations",
		withAliases("invit", "inv", "i"),
		withChildCommands(
			newOrganizationsInvitationsSendCommand(),
			newOrganizationsInvitationsListCommand(),
		),
	)
}
