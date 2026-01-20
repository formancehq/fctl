package webhooks

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("webhooks",
		fctl.WithAliases("web", "wh"),
		fctl.WithShortDescription("Webhooks management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewDeactivateCommand(),
			NewActivateCommand(),
			NewDeleteCommand(),
			NewChangeSecretCommand(),
		),
	)
}
