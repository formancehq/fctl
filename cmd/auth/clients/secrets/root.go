package secrets

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("secrets",
		fctl.WithAliases("sec"),
		fctl.WithShortDescription("Secrets management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewDeleteCommand(),
		),
	)
}
