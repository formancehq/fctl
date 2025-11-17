package payments

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewPaymentsCommand() *cobra.Command {
	return fctl.NewCommand("payments",
		fctl.WithAliases("p"),
		fctl.WithShortDescription("Payments management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
		),
	)
}
