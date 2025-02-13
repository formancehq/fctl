package logs

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewLogsCommand() *cobra.Command {
	return fctl.NewCommand("logs",
		fctl.WithAliases("log", "l"),
		fctl.WithShortDescription("Logs management"),
		fctl.WithChildCommands(
			NewListCommand(),
		),
	)
}
