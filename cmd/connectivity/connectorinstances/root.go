package connectorinstances

import (
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(factory connectivityinternal.ClientFactory, read ReadFileFunc, paths PathCompleter) *cobra.Command {
	return fctl.NewCommand(
		"connectorinstances",
		fctl.WithAliases("connectorinstance", "instances", "instance", "ci"),
		fctl.WithShortDescription("Manage Connectivity connector instances"),
		fctl.WithChildCommands(
			NewListCommand(factory),
			NewShowCommand(factory),
			NewInstallCommand(factory, read, paths),
			NewConfigureCommand(factory, read, paths),
			NewSuspendCommand(factory),
			NewUnsuspendCommand(factory),
			NewUninstallCommand(factory),
		),
	)
}
