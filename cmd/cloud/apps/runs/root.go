package runs

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("runs",
		fctl.WithShortDescription("Manage app runs"),
		fctl.WithChildCommands(
			NewList(),
			NewShow(),
			NewLogs(),
			// NewWait(),
		),
	)
}
