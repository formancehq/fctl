package clients

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/auth/clients/secrets"
	"github.com/formancehq/fctl/v3/cmd/auth/users"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("clients",
		fctl.WithAliases("client", "c"),
		fctl.WithShortDescription("Clients management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
			NewUpdateCommand(),
			NewShowCommand(),
			secrets.NewCommand(),
			users.NewCommand(),
		),
	)
}
