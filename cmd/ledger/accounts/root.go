package accounts

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewLedgerAccountsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("accounts",
		cmdbuilder.WithAliases("acc", "a", "ac", "account"),
		cmdbuilder.WithShortDescription("Accounts management"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
		),
	)
}
