package accounts

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewLedgerAccountsCommand() *cobra.Command {
	return fctl.NewCommand("accounts",
		fctl.WithAliases("acc", "a", "ac", "account"),
		fctl.WithShortDescription("Accounts management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewSetMetadataCommand(),
			NewDeleteMetadataCommand(),
		),
	)
}
