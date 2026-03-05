package plugin

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
	pluginpkg "github.com/formancehq/fctl/pkg/plugin"
)

const allFlag = "all"

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand("update",
		fctl.WithShortDescription("Update plugins to the latest version"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithBoolFlag(allFlag, false, "Update all installed plugins"),
		fctl.WithRunE(runUpdate),
	)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	configDir := fctl.GetString(cmd, fctl.ConfigDir)
	updateAll := fctl.GetBool(cmd, allFlag)

	cfg, err := pluginpkg.LoadPluginsConfig(configDir)
	if err != nil {
		return err
	}

	registry := pluginpkg.NewRegistryClient(fctl.GetHttpClient(cmd))
	pm := pluginpkg.NewPluginManager(configDir)

	var toUpdate []string
	if updateAll {
		for _, p := range cfg.Plugins {
			toUpdate = append(toUpdate, p.Name)
		}
	} else if len(args) > 0 {
		toUpdate = []string{args[0]}
	} else {
		return fmt.Errorf("specify a plugin name or use --all to update all plugins")
	}

	for _, name := range toUpdate {
		pterm.Info.Printfln("Updating plugin %s...", name)
		if err := pm.InstallPlugin(name, "", registry); err != nil {
			pterm.Error.Printfln("Failed to update plugin %s: %v", name, err)
			continue
		}
		pterm.Success.Printfln("Plugin %s updated", name)
	}

	return nil
}
