package install

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	return internal.NewCommand("install",
		internal.WithAliases("i"),
		internal.WithShortDescription("Install a connector"),
		internal.WithChildCommands(
			NewStripeCommand(),
		),
	)
}
