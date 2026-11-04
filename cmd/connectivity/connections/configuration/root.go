package configuration

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(update *cobra.Command) *cobra.Command {
	return fctl.NewCommand("config",
		fctl.WithAliases("configuration"),
		fctl.WithShortDescription("Manage Connectivity connection configuration"),
		fctl.WithChildCommands(update),
	)
}
