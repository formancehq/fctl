package accounts

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewAccountsCommand() *cobra.Command {
	return fctl.NewCommand("accounts",
		fctl.WithAliases("acc", "a", "ac", "account"),
		fctl.WithShortDescription("Accounts management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewListBalanceCommand(),
		),
	)
}
