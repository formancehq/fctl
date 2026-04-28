package users

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("users",
		fctl.WithAliases("u", "user"),
		fctl.WithShortDescription("Manage users"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewLinkCommand(),
			NewUnlinkCommand(),
		),
	)
}
