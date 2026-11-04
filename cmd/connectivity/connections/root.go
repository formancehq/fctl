package connections

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(factory connectivityinternal.ClientFactory, read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	update := NewConfigureCommand(factory, read, paths)
	update.Use = "update <connection>"
	update.Aliases = nil
	configure := NewConfigureCommand(factory, read, paths)
	return fctl.NewCommand(
		"connections",
		fctl.WithAliases("connection"),
		fctl.WithShortDescription("Manage Connectivity connections"),
		fctl.WithChildCommands(
			NewListCommand(factory),
			NewShowCommand(factory),
			NewInstallCommand(factory, read, paths),
			configuration.NewCommand(update),
			configure,
			NewSuspendCommand(factory),
			NewUnsuspendCommand(factory),
			NewUninstallCommand(factory),
		),
	)
}
