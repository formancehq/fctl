package secrets

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("secrets",
		fctl.WithAliases("sec"),
		fctl.WithShortDescription("Manage secrets"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewDeleteCommand(),
		),
	)
}
