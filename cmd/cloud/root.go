package cloud

import (
	"github.com/formancehq/fctl/cmd/cloud/me"
	"github.com/formancehq/fctl/cmd/cloud/organizations"
	"github.com/formancehq/fctl/cmd/cloud/users"
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("cloud",
		internal.WithAliases("c"),
		internal.WithShortDescription("Cloud management"),
		internal.WithChildCommands(
			organizations.NewCommand(),
			me.NewCommand(),
			users.NewCommand(),
			NewGeneratePersonalTokenCommand(),
		),
	)
}
