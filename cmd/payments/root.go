package payments

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/payments/connectors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("payments",
		cmdbuilder.WithShortDescription("Payments management"),
		cmdbuilder.WithChildCommands(
			connectors.NewConnectorsCommand(),
		),
	)
}
