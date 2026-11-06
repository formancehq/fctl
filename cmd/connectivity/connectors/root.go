package connectors

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand(
		"connectors",
		fctl.WithAliases("connector", "c"),
		fctl.WithShortDescription("Browse Connectivity connectors"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
