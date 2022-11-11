package ledger

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/accounts"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/transactions"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("ledger",
		cmdbuilder.WithPersistentStringFlag(internal.LedgerFlag, "default", "Specific ledger"),
		cmdbuilder.WithChildCommands(
			transactions.NewLedgerTransactionsCommand(),
			NewLedgerBalancesCommand(),
			accounts.NewLedgerAccountsCommand(),
		),
	)
}
