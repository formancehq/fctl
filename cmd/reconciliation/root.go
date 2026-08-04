package reconciliation

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/reconciliation/alerts"
	"github.com/formancehq/fctl/v3/cmd/reconciliation/evaluations"
	"github.com/formancehq/fctl/v3/cmd/reconciliation/policies"
	"github.com/formancehq/fctl/v3/cmd/reconciliation/rules"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("reconciliation",
		fctl.WithShortDescription("Manage reconciliation"),
		fctl.WithChildCommands(
			policies.NewPoliciesCommand(),
			rules.NewCommand(),
			evaluations.NewCommand(),
			alerts.NewCommand(),
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
