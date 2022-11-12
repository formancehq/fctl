package webhooks

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("webhooks",
		cmdbuilder.WithAliases("web", "wh"),
		cmdbuilder.WithShortDescription("Webhooks management"),
		cmdbuilder.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewDeactivateCommand(),
			NewActivateCommand(),
			NewDeleteCommand(),
			NewChangeSecretCommand(),
		),
	)
}
