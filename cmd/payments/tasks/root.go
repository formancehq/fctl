package tasks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
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
