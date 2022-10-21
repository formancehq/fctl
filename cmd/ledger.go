package cmd

import (
	"github.com/spf13/cobra"
)

const (
	ledgerFlag = "ledger"
)

func newLedgerCommand() *cobra.Command {
	return newStackCommand("ledger",
		withPersistentStringFlag(ledgerFlag, "default", "Specific ledger"),
		withChildCommands(
			newLedgerTransactionsCommand(),
			newLedgerBalancesCommand(),
			newLedgerAccountsCommand(),
		),
	)
}
