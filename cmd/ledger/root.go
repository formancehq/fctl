package ledger

import (
	"github.com/formancehq/fctl/cmd/ledger/accounts"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/fctl/cmd/ledger/transactions"
	internal2 "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal2.NewStackCommand("ledger",
		internal2.WithAliases("l"),
		internal2.WithPersistentStringFlag(internal.LedgerFlag, "default", "Specific ledger"),
		internal2.WithShortDescription("Ledger management"),
		internal2.WithChildCommands(
			NewBalancesCommand(),
			NewSendCommand(),
			NewStatsCommand(),
			NewServerInfoCommand(),
			transactions.NewLedgerTransactionsCommand(),
			accounts.NewLedgerAccountsCommand(),
		),
	)
}
