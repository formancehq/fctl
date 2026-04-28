package transactions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewLedgerTransactionsCommand() *cobra.Command {
	return fctl.NewCommand("transactions",
		fctl.WithAliases("t", "txs", "tx"),
		fctl.WithShortDescription("Transactions management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewNumCommand(),
			NewRevertCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
			NewDeleteMetadataCommand(),
		),
	)
}
