package conversions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewConversionsCommand() *cobra.Command {
	return fctl.NewCommand("conversions",
		fctl.WithAliases("cv"),
		fctl.WithShortDescription("Manage currency conversions (read-only) ingested from exchange-style connectors"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
		),
	)
}
