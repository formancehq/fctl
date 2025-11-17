package pools

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewPoolsCommand() *cobra.Command {
	return fctl.NewCommand("pools",
		fctl.WithAliases("p"),
		fctl.WithShortDescription("Pools management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
			NewBalancesCommand(),
			NewAddAccountCommand(),
			NewRemoveAccountCommand(),
		),
	)
}
