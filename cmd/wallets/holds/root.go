package holds

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("holds",
		fctl.WithAliases("h", "hold"),
		fctl.WithShortDescription("Wallets holds management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewVoidCommand(),
			NewConfirmCommand(),
			NewShowCommand(),
		),
	)
}
