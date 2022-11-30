package invitations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewStackCommand("invitations",
		internal.WithAliases("invit", "inv", "i"),
		internal.WithShortDescription("Invitations management"),
		internal.WithChildCommands(
			NewSendCommand(),
			NewListCommand(),
		),
	)
}
