package variables

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("variables",
		fctl.WithShortDescription("Manage app variables"),
		fctl.WithAliases("var", "vars"),
		fctl.WithChildCommands(
			NewList(),
			NewCreate(),
			NewDelete(),
		),
	)
}
