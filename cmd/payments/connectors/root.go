package connectors

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/configs"
	"github.com/formancehq/fctl/v3/cmd/payments/connectors/install"
	"github.com/formancehq/fctl/v3/cmd/payments/connectors/schedules"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewConnectorsCommand() *cobra.Command {
	return fctl.NewCommand("connectors",
		fctl.WithAliases("c", "co", "con"),
		fctl.WithShortDescription("Manage connectors"),
		fctl.WithChildCommands(
			NewUninstallCommand(),
			NewListCommand(),
			install.NewInstallCommand(),
			configs.NewUpdateConfigCommand(),
			configs.NewGetConfigCommand(),
			NewConnectorListAvailableCommand(),
			schedules.NewSchedulesCommand(),
		),
	)
}
