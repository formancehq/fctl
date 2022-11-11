package cloud

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/cloud/me"
	"github.com/formancehq/fctl/cmd/subcmds/cloud/organizations"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("cloud",
		cmdbuilder.WithAliases("c"),
		cmdbuilder.WithChildCommands(
			organizations.NewCommand(),
			me.NewCommand(),
		),
	)
}
