package organizations

import (
	"github.com/formancehq/fctl/cmd/cloud/organizations/invitations"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("organizations",
		cmdbuilder.WithAliases("org", "o"),
		cmdbuilder.WithShortDescription("Organizations management"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			invitations.NewCommand(),
		),
	)
}
