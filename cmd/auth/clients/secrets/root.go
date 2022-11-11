package secrets

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewAuthClientsSecretsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("secrets",
		cmdbuilder.WithChildCommands(
			NewAuthClientsSecretsCreateCommand(),
			NewAuthClientsSecretsDeleteCommand(),
		),
	)
}
