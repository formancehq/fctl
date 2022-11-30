package clients

import (
	"github.com/formancehq/fctl/cmd/auth/clients/secrets"
	"github.com/formancehq/fctl/cmd/auth/users"
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("clients",
		internal.WithAliases("client", "c"),
		internal.WithShortDescription("Clients management"),
		internal.WithChildCommands(
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
