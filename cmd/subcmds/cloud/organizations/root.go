package organizations

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/cloud/organizations/invitations"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("organizations",
		cmdbuilder.WithAliases("org", "o"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			invitations.NewOrganizationsInvitationsCommand(),
		),
	)
}
