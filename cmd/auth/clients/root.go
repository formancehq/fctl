package clients

import (
	"github.com/formancehq/fctl/cmd/auth/clients/secrets"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewAuthClientsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("clients",
		cmdbuilder.WithChildCommands(
			NewAuthClientsListCommand(),
			NewAuthClientsCreateCommand(),
			NewAuthClientsDeleteCommand(),
			NewAuthClientsUpdateCommand(),
			secrets.NewAuthClientsSecretsCommand(),
			NewAuthClientsShowCommand(),
		),
	)
}
