package secrets

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("secrets",
		cmdbuilder.WithAliases("sec"),
		cmdbuilder.WithShortDescription("Secrets management"),
		cmdbuilder.WithChildCommands(
			NewCreateCommand(),
			NewDeleteCommand(),
		),
	)
}
