package transactions

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewLedgerTransactionsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("transactions",
		cmdbuilder.WithChildCommands(
			NewLedgerTransactionsListCommand(),
			NewLedgerTransactionsNumscriptCommand(),
			NewLedgerTransactionsRevertCommand(),
			NewLedgerTransactionsShowCommand(),
		),
	)
}