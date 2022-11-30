package accounts

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewLedgerAccountsCommand() *cobra.Command {
	return internal.NewCommand("accounts",
		internal.WithAliases("acc", "a", "ac", "account"),
		internal.WithShortDescription("Accounts management"),
		internal.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
		),
	)
}
