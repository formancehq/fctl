package schedules

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/schedules/instances"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewSchedulesCommand() *cobra.Command {
	return fctl.NewCommand("schedules",
		fctl.WithAliases("sch"),
		fctl.WithShortDescription("Manage connector schedules"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			instances.NewInstancesCommand(),
		),
	)
}
