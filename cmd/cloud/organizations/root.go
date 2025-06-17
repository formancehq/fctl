package organizations

import (
	authorization_provider "github.com/formancehq/fctl/cmd/cloud/organizations/authentication-provider"
	"github.com/formancehq/fctl/cmd/cloud/organizations/invitations"
	"github.com/formancehq/fctl/cmd/cloud/organizations/oauth"
	"github.com/formancehq/fctl/cmd/cloud/organizations/users"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
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
			authorization_provider.NewCommand(),
		),
	)
}
