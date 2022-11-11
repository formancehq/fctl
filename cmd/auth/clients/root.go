package clients

import (
	"github.com/formancehq/fctl/cmd/auth/clients/secrets"
	"github.com/formancehq/fctl/cmd/auth/users"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("clients",
		cmdbuilder.WithAliases("client", "c"),
		cmdbuilder.WithShortDescription("Clients management"),
		cmdbuilder.WithChildCommands(
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
