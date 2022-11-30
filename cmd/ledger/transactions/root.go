package transactions

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewLedgerTransactionsCommand() *cobra.Command {
	return internal.NewCommand("transactions",
		internal.WithAliases("t", "txs", "tx"),
		internal.WithShortDescription("Transactions management"),
		internal.WithChildCommands(
			NewListCommand(),
			NewCommand(),
			NewRevertCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
		),
	)
}
