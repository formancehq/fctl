package organizations

import (
	"github.com/formancehq/fctl/cmd/cloud/organizations/invitations"
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewStackCommand("organizations",
		internal.WithAliases("org", "o"),
		internal.WithShortDescription("Organizations management"),
		internal.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
			invitations.NewCommand(),
		),
	)
}
