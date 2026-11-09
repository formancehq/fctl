package configuration

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(update *cobra.Command) *cobra.Command {
	return fctl.NewCommand("config",
		fctl.WithAliases("configuration"),
		fctl.WithShortDescription("Alias group for connections configure"),
		// configure, and its update alias, already cover this group.
		fctl.WithHidden(),
		fctl.WithChildCommands(update),
	)
}
