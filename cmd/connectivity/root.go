package connectivity

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections"
	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("connectivity",
		fctl.WithShortDescription("Manage Connectivity connectors and connections"),
		fctl.WithChildCommands(
			NewStatusCommand(),
			connectors.NewCommand(),
			connections.NewCommand(fctl.ReadFile, connections.OSPathCompleter),
		),
	)
}
