package versions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("versions",
		fctl.WithShortDescription("Manage manifest versions"),
		fctl.WithAliases("ver", "v"),
		fctl.WithChildCommands(
			NewList(),
			NewShow(),
			NewPush(),
		),
	)
}
