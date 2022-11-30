package auth

import (
	"github.com/formancehq/fctl/cmd/auth/clients"
	"github.com/formancehq/fctl/cmd/auth/users"
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewStackCommand("auth",
		internal.WithShortDescription("Auth server management"),
		internal.WithChildCommands(
			clients.NewCommand(),
			users.NewCommand(),
		),
	)
}
