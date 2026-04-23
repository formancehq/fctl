package payments

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewPaymentsCommand() *cobra.Command {
	return fctl.NewCommand("payments",
		fctl.WithAliases("p"),
		fctl.WithShortDescription("Manage payments"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
		),
	)
}
