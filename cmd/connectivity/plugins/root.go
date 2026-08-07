package plugins

import (
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	return fctl.NewCommand(
		"plugins",
		fctl.WithAliases("plugin", "p"),
		fctl.WithShortDescription("Browse Connectivity plugins"),
		fctl.WithChildCommands(
			NewListCommand(factory),
			NewShowCommand(factory),
		),
	)
}
