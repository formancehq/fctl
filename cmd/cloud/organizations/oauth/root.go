package oauth

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("oauth",
		fctl.WithShortDescription("client management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
		),
	)
}
