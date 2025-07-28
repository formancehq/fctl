package oauth_clients

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("oauth-clients",
		fctl.WithShortDescription("Oauth clients management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
		),
	)
}
