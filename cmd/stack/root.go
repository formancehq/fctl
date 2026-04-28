package stack

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/stack/modules"
	"github.com/formancehq/fctl/v3/cmd/stack/users"
	fctl "github.com/formancehq/fctl/v3/pkg"
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
	)
}
