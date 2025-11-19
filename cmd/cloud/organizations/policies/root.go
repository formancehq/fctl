package policies

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("policies",
		fctl.WithShortDescription("Policies management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewShowCommand(),
			NewUpdateCommand(),
			NewDeleteCommand(),
			NewAddScopeCommand(),
			NewRemoveScopeCommand(),
		),
	)
}
