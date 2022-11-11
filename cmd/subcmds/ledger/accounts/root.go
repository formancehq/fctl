package accounts

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewLedgerAccountsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("accounts",
		cmdbuilder.WithShortDescription("handle ledger accounts"),
		cmdbuilder.WithChildCommands(
			NewLedgerAccountsListCommand(),
			NewLedgerAccountsShowCommand(),
		),
	)
}
