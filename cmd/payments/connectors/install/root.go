package install

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewInstallCommand() *cobra.Command {
	c := NewConnectorInstallController()
	return fctl.NewCommand("install <connector> <file>|-",
		fctl.WithAliases("i"),
		fctl.WithShortDescription("Install a connector"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ConnectorInstallStore](c),
	)
}
