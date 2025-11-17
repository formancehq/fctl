package wallets

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/wallets/balances"
	"github.com/formancehq/fctl/cmd/wallets/holds"
	"github.com/formancehq/fctl/cmd/wallets/transactions"
	fctl "github.com/formancehq/fctl/pkg"
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
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return fctl.NewStackStore(cmd)
		}),
	)
}
