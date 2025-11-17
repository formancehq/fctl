package reconciliation

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/reconciliation/policies"
	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("reconciliation",
		fctl.WithShortDescription("Reconciliation management"),
		fctl.WithChildCommands(
			policies.NewPoliciesCommand(),
			NewListCommand(),
			NewShowCommand(),
		),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return fctl.NewStackStore(cmd)
		}),
	)
}
