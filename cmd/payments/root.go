package payments

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/payments/connectors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewStackCommand("payments",
		internal.WithShortDescription("Payments management"),
		internal.WithChildCommands(
			connectors.NewConnectorsCommand(),
		),
	)
}
