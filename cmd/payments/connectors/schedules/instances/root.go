package instances

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewInstancesCommand() *cobra.Command {
	return fctl.NewCommand("instances",
		fctl.WithAliases("inst"),
		fctl.WithShortDescription("Manage connector schedule instances"),
		fctl.WithChildCommands(
			NewListCommand(),
		),
	)
}
