package connections

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand(read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	update := NewConfigureCommand(read, paths)
	update.Use = "update <connection>"
	update.Aliases = nil
	configure := NewConfigureCommand(read, paths)
	return fctl.NewCommand(
		"connections",
		fctl.WithAliases("connection"),
		fctl.WithShortDescription("Install and manage connections"),
		fctl.WithChildCommands(
			NewListCommand(),
			NewShowCommand(),
			NewInstallCommand(read, paths),
			configuration.NewCommand(update),
			configure,
			NewSuspendCommand(),
			NewResumeCommand(),
			NewUninstallCommand(),
		),
	)
}
