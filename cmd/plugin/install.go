package plugin

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
	pluginpkg "github.com/formancehq/fctl/pkg/plugin"
)

const versionFlag = "version"

func NewInstallCommand() *cobra.Command {
	return fctl.NewCommand("install",
		fctl.WithShortDescription("Install a plugin from the registry"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag(versionFlag, "", "Specific version to install (defaults to latest)"),
		fctl.WithRunE(runInstall),
	)
}

func runInstall(cmd *cobra.Command, args []string) error {
	name := args[0]
	version := fctl.GetString(cmd, versionFlag)
	configDir := fctl.GetString(cmd, fctl.ConfigDir)

	registry := pluginpkg.NewRegistryClient(fctl.GetHttpClient(cmd))
	pm := pluginpkg.NewPluginManager(configDir)

	pterm.Info.Printfln("Installing plugin %s...", name)

	if err := pm.InstallPlugin(name, version, registry); err != nil {
		return fmt.Errorf("failed to install plugin %s: %w", name, err)
	}

	pterm.Success.Printfln("Plugin %s installed successfully", name)
	return nil
}
