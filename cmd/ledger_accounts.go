package cmd

import (
	"github.com/spf13/cobra"
)

func newLedgerAccountsCommand() *cobra.Command {
	return newCommand("accounts",
		withShortDescription("handle ledger accounts"),
		withChildCommands(
			newLedgerAccountsListCommand(),
			newLedgerAccountsShowCommand(),
		),
	)
}
