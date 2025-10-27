package runs

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
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
