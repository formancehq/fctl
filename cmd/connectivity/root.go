package connectivity

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connectorinstances"
	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	factory := connectivityinternal.NewClientFactory()
	return fctl.NewStackCommand("connectivity",
		fctl.WithShortDescription("Manage Connectivity connectors and connector instances"),
		fctl.WithChildCommands(
			connectors.NewCommand(factory),
			connectorinstances.NewCommand(factory, fctl.ReadFile, connectorinstances.OSPathCompleter),
		),
	)
}
