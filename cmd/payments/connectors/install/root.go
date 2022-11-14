package install

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	return cmdbuilder.NewCommand("install",
		cmdbuilder.WithAliases("i"),
		cmdbuilder.WithShortDescription("Install a connector"),
		cmdbuilder.WithChildCommands(
			NewStripeCommand(),
		),
	)
}
