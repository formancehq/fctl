package orders

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewOrdersCommand() *cobra.Command {
	return fctl.NewCommand("orders",
		fctl.WithAliases("o"),
		fctl.WithShortDescription("Manage orders (read-only) ingested from exchange-style connectors"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
