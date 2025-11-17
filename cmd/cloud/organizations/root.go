package organizations

import (
	"github.com/spf13/cobra"

	authorization_provider "github.com/formancehq/fctl/cmd/cloud/organizations/authentication-provider"
	"github.com/formancehq/fctl/cmd/cloud/organizations/invitations"
	"github.com/formancehq/fctl/cmd/cloud/organizations/oauth"
	oauth_clients "github.com/formancehq/fctl/cmd/cloud/organizations/oauth-clients"
	"github.com/formancehq/fctl/cmd/cloud/organizations/users"
	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("organizations",
		fctl.WithAliases("org", "o", "organization"),
		fctl.WithShortDescription("Organizations management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
			NewUpdateCommand(),
			NewDescribeCommand(),
			NewHistoryCommand(),
			users.NewCommand(),
			invitations.NewCommand(),
			oauth.NewCommand(),
			oauth_clients.NewCommand(),
			authorization_provider.NewCommand(),
		),
	)
}
