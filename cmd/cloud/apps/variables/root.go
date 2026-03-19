package variables

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
