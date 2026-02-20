package instances

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("instances",
		fctl.WithAliases("ins", "i"),
		fctl.WithShortDescription("Instances management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewDescribeCommand(),
			NewSendEventCommand(),
			NewStopCommand(),
		),
	)
}
