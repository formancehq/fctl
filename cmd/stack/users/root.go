package users

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
