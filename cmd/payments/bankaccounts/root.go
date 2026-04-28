package bankaccounts

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewBankAccountsCommand() *cobra.Command {
	return fctl.NewCommand("bank_accounts",
		fctl.WithAliases("bacc", "ba", "bac", "baccount"),
		fctl.WithShortDescription("Manage bank accounts"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewForwardCommand(),
			NewUpdateMetadataCommand(),
			NewShowCommand(),
			NewListCommand(),
		),
	)
}
