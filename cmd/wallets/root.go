package wallets

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/wallets/balances"
	"github.com/formancehq/fctl/v3/cmd/wallets/holds"
	"github.com/formancehq/fctl/v3/cmd/wallets/transactions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("wallets",
		fctl.WithAliases("wal", "wa", "wallet"),
		fctl.WithShortDescription("Wallets management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewUpdateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewCreditWalletCommand(),
			NewDebitWalletCommand(),
			transactions.NewCommand(),
			holds.NewCommand(),
			balances.NewCommand(),
		),
	)
}
