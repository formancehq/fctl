package cloud

import (
	"github.com/formancehq/fctl/cmd/cloud/me"
	"github.com/formancehq/fctl/cmd/cloud/organizations"
	"github.com/formancehq/fctl/cmd/cloud/users"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("cloud",
		cmdbuilder.WithAliases("c"),
		cmdbuilder.WithShortDescription("Cloud management"),
		cmdbuilder.WithChildCommands(
			organizations.NewCommand(),
			me.NewCommand(),
			users.NewCommand(),
		),
	)
}
