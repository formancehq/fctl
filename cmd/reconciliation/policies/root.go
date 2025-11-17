package policies

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewPoliciesCommand() *cobra.Command {
	return fctl.NewCommand("policies",
		fctl.WithAliases("p"),
		fctl.WithShortDescription("Policies management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
			NewReconciliationCommand(),
		),
	)
}
