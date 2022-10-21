package cmd

import (
	"github.com/spf13/cobra"
)

func newLedgerTransactionsCommand() *cobra.Command {
	return newCommand("transactions",
		withChildCommands(
			newLedgerTransactionsListCommand(),
			newLedgerTransactionsNumscriptCommand(),
			newLedgerTransactionsRevertCommand(),
			newLedgerTransactionsShowCommand(),
		),
	)
}
