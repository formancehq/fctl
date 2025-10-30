package users

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("users",
		fctl.WithAliases("u", "user"),
		fctl.WithShortDescription("Stack users management within an organization"),
		fctl.WithChildCommands(
			NewLinkCommand(),
			NewListCommand(),
			NewUnlinkCommand(),
		),
	)
}
