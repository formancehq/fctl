package connectors

import (
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	return fctl.NewCommand(
		"connectors",
		fctl.WithAliases("connector", "c"),
		fctl.WithShortDescription("Browse Connectivity connectors"),
		fctl.WithChildCommands(
			NewListCommand(factory),
			NewShowCommand(factory),
		),
	)
}
