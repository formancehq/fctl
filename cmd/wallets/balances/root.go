package balances

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("balances",
		fctl.WithAliases("balance", "bls", "bal"),
		fctl.WithShortDescription("Wallet balances"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewCreateCommand(),
		),
	)
}
