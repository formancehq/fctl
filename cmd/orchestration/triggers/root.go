package triggers

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/orchestration/triggers/occurrences"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("triggers",
		fctl.WithAliases("trig", "t"),
		fctl.WithShortDescription("Triggers management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
			NewCreateCommand(),
			NewTestCommand(),
			occurrences.NewCommand(),
		),
	)
}
