package payments

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/payments/connectors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("payments",
		cmdbuilder.WithChildCommands(
			connectors.NewPaymentsConnectorsCommand(),
		),
	)
}
