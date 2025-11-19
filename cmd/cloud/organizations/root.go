package organizations

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/cloud/organizations/applications"
	authorization_provider "github.com/formancehq/fctl/cmd/cloud/organizations/authentication-provider"
	"github.com/formancehq/fctl/cmd/cloud/organizations/invitations"
	oauth_clients "github.com/formancehq/fctl/cmd/cloud/organizations/oauth-clients"
	"github.com/formancehq/fctl/cmd/cloud/organizations/policies"
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
			oauth_clients.NewCommand(),
			authorization_provider.NewCommand(),
			policies.NewCommand(),
			applications.NewCommand(),
		),
	)
}
