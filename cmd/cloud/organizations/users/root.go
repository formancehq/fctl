package users

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("users",
		fctl.WithAliases("u", "user"),
		fctl.WithShortDescription("Users management"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewLinkCommand(),
			NewUnlinkCommand(),
		),
	)
}
