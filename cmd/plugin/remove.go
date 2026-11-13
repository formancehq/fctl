package plugin

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	pluginpkg "github.com/formancehq/fctl/v3/pkg/plugin"
)

func NewRemoveCommand() *cobra.Command {
	return fctl.NewCommand("remove",
		fctl.WithAliases("rm", "uninstall"),
		fctl.WithShortDescription("Remove an installed plugin"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithRunE(runRemove),
	)
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	configDir := fctl.GetString(cmd, fctl.ConfigDir)

	pm := pluginpkg.NewPluginManager(configDir)

	if err := pm.RemovePlugin(name); err != nil {
		return err
	}

	pterm.Success.Printfln("Plugin %s removed", name)
	return nil
}
