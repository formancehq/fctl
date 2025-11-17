package stack

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/stack/modules"
	"github.com/formancehq/fctl/cmd/stack/users"
	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("stack",
		fctl.WithShortDescription("Manage your stack"),
		fctl.WithAliases("stack", "stacks", "st"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewDeleteCommand(),
			NewShowCommand(),
			NewDisableCommand(),
			NewEnableCommand(),
			NewRestoreStackCommand(),
			NewUpdateCommand(),
			NewUpgradeCommand(),
			NewHistoryCommand(),
			NewProxyCommand(),
			users.NewCommand(),
			modules.NewCommand(),
		),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return fctl.NewMembershipOrganizationStore(cmd)
		}),
	)
}
