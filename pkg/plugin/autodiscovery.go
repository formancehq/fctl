package plugin

import (
	"context"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// AutoDiscover handles the NeedInstall resolution by prompting the user to
// install the required plugin. In non-interactive mode, it returns an error
// with an actionable message.
//
// On success, it returns the newly loaded plugin ready for use.
func AutoDiscover(
	ctx context.Context,
	need NeedInstall,
	manager *PluginManager,
	registry *RegistryClient,
) (*LoadedPlugin, error) {
	if !isInteractive() {
		return nil, fmt.Errorf(
			"stack runs %s v%s which requires the %q plugin (v%s)\n\n  Run: fctl plugin install %s --version %s",
			need.ServiceName, need.ServiceVersion,
			need.ServiceName, need.PluginVersion,
			need.ServiceName, need.PluginVersion,
		)
	}

	// Interactive prompt
	pterm.Println()
	pterm.Info.Printfln(
		"Stack runs %s v%s. Compatible plugin: fctl-plugin-%s v%s",
		need.ServiceName, need.ServiceVersion,
		need.ServiceName, need.PluginVersion,
	)

	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show("Install it now?")

	if !result {
		return nil, fmt.Errorf("plugin installation declined")
	}

	pterm.Info.Printfln("Downloading fctl-plugin-%s v%s...", need.ServiceName, need.PluginVersion)

	if err := manager.InstallFromRegistry(need.ServiceName, need.PluginVersion, *need.RegistryPlugin, registry); err != nil {
		return nil, fmt.Errorf("failed to install plugin: %w", err)
	}

	pterm.Success.Printfln("Plugin installed.")
	pterm.Println()

	// Load the freshly installed plugin
	binaryPath := PluginBinaryPath(manager.configDir, need.ServiceName, need.PluginVersion)
	loaded, err := LoadPlugin(need.ServiceName, binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load installed plugin: %w", err)
	}
	loaded.Version = need.PluginVersion
	if entry, ok := need.RegistryPlugin.Versions[need.PluginVersion]; ok {
		loaded.CompatibleWith = entry.CompatibleWith
	}

	return loaded, nil
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
