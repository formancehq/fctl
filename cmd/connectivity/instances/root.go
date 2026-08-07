package instances

import (
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(factory connectivityinternal.ClientFactory, read ReadFileFunc, paths PathCompleter) *cobra.Command {
	return fctl.NewCommand(
		"instances",
		fctl.WithAliases("instance", "i"),
		fctl.WithShortDescription("Manage Connectivity instances"),
		fctl.WithChildCommands(
			NewListCommand(factory),
			NewShowCommand(factory),
			NewInstallCommand(factory, read, paths),
			NewConfigureCommand(factory, read, paths),
			NewUninstallCommand(factory),
		),
	)
}
