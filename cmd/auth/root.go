package auth

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/auth/clients"
	"github.com/formancehq/fctl/v3/cmd/auth/users"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("auth",
		fctl.WithShortDescription("Auth server management"),
		fctl.WithChildCommands(
			clients.NewCommand(),
			users.NewCommand(),
		),
	)
}
