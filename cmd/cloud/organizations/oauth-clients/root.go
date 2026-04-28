package oauth_clients

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("oauth-clients",
		fctl.WithShortDescription("Oauth clients management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
			NewUpdateCommand(),
		),
	)
}
