package reconciliation

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/reconciliation/policies"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("reconciliation",
		fctl.WithShortDescription("Reconciliation management"),
		fctl.WithChildCommands(
			policies.NewPoliciesCommand(),
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
