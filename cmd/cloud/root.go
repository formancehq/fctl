package cloud

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/cloud/apps"
	"github.com/formancehq/fctl/cmd/cloud/me"
	"github.com/formancehq/fctl/cmd/cloud/organizations"
	"github.com/formancehq/fctl/cmd/cloud/regions"
	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("cloud",
		fctl.WithAliases("c"),
		fctl.WithShortDescription("Cloud management"),
		fctl.WithChildCommands(
			organizations.NewCommand(),
			me.NewCommand(),
			regions.NewCommand(),
			NewGeneratePersonalTokenCommand(),
			apps.NewCommand(),
		),
	)
}
