package tasks

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewTasksCommand() *cobra.Command {
	return fctl.NewCommand("tasks",
		fctl.WithAliases("t"),
		fctl.WithShortDescription("Tasks management"),
		fctl.WithChildCommands(
			NewShowCommand(),
		),
	)
}
