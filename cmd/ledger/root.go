package ledger

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/ledger/accounts"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/fctl/cmd/ledger/transactions"
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
