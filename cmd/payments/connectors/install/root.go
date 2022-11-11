package install

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewPaymentsConnectorsInstallCommand() *cobra.Command {
	return cmdbuilder.NewCommand("install",
		cmdbuilder.WithChildCommands(
			NewPaymentsConnectorsInstallStripeCommand(),
		),
	)
}
