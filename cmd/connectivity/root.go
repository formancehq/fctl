package connectivity

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/instances"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	"github.com/formancehq/fctl/v3/cmd/connectivity/plugins"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	factory := connectivityinternal.NewClientFactory()
	return fctl.NewStackCommand("connectivity",
		fctl.WithShortDescription("Manage Connectivity plugins and instances"),
		fctl.WithChildCommands(
			plugins.NewCommand(factory),
			instances.NewCommand(factory, fctl.ReadFile, instances.OSPathCompleter),
		),
	)
}
