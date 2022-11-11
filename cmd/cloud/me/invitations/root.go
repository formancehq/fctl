package invitations

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("invitations",
		cmdbuilder.WithShortDescription("Invitations management"),
		cmdbuilder.WithAliases("invit", "i"),
		cmdbuilder.WithChildCommands(
			NewListCommand(),
			NewAcceptCommand(),
			NewDeclineCommand(),
		),
	)
}
