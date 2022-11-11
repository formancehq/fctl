package clients

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/auth/clients/secrets"
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
