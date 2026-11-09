package connectivity

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections"
	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("connectivity",
		fctl.WithShortDescription("Manage connectors and connections on the Connectivity service"),
		fctl.WithExamples(
			"fctl connectivity connectors list --filter tags=provider:psp",
			"fctl connectivity connections show stripeconnect-fpss",
		),
		fctl.WithChildCommands(
			NewServerInfosCommand(),
			connectors.NewCommand(),
			connections.NewCommand(fctl.ReadFile, connections.OSPathCompleter),
		),
	)
}
