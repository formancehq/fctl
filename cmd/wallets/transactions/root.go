package transactions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("transactions",
		fctl.WithAliases("transaction", "tx", "txs"),
		fctl.WithShortDescription("Wallet transactions"),
		fctl.WithChildCommands(
			NewListCommand(),
		),
	)
}
