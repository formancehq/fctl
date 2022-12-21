package wallet

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("wallet",
		fctl.WithAliases("wal", "wa"),
		fctl.WithShortDescription("Wallets management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
		),
	)
}
