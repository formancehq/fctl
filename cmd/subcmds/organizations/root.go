package organizations

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/organizations/invitations"
	"github.com/spf13/cobra"
)

func NewOrganizationsCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("organizations",
		cmdbuilder.WithAliases("org", "o"),
		cmdbuilder.WithChildCommands(
			NewOrganizationsListCommand(),
			invitations.NewOrganizationsInvitationsCommand(),
		),
	)
}
