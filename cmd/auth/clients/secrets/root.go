package secrets

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("secrets",
		internal.WithAliases("sec"),
		internal.WithShortDescription("Secrets management"),
		internal.WithChildCommands(
			NewCreateCommand(),
			NewDeleteCommand(),
		),
	)
}
