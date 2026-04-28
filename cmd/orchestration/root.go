package orchestration

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/orchestration/instances"
	"github.com/formancehq/fctl/v3/cmd/orchestration/triggers"
	"github.com/formancehq/fctl/v3/cmd/orchestration/workflows"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("orchestration",
		fctl.WithAliases("orch", "or"),
		fctl.WithShortDescription("Orchestration"),
		fctl.WithHidden(),
		fctl.WithChildCommands(
			instances.NewCommand(),
			workflows.NewCommand(),
			triggers.NewCommand(),
		),
	)
}
