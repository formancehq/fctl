package oauth

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("oauth",
		fctl.WithShortDescription("client management"),
		fctl.WithDeprecated("Use `fctl cloud organizations clients` instead"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
		),
	)
}
