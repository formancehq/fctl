package pluginhost

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

// Install validates and caches a trusted local service executable.
func Install(cmd *cobra.Command, settings *connection.Settings, service, binary, localVersion string) (pluginmanager.Lock, error) {
	if err := validateService(service); err != nil {
		return pluginmanager.Lock{}, err
	}
	if binary == "" {
		return pluginmanager.Lock{}, fmt.Errorf("provide --binary PATH to the %s plugin executable", service)
	}
	path, err := filepath.Abs(binary)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	manifest, err := readPluginManifest(cmd.Context(), path)
	if err != nil {
		return pluginmanager.Lock{}, err
	}

	if err := validateServiceManifest(cmd.Context(), cmd.Root(), service, manifest); err != nil {
		return pluginmanager.Lock{}, err
	}
	target, version, err := pluginServiceSyncTarget(cmd.Context(), settings, cmd, service, localVersion)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	if manifest.Version != version {
		return pluginmanager.Lock{}, fmt.Errorf("plugin targets %s %s, selected service is %s; build the matching plugin", service, manifest.Version, version)
	}
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	return manager.InstallLocal(cmd.Context(), path, target, manifest)
}

// Sync prepares the plugin for the exact selected service version.
func Sync(cmd *cobra.Command, settings *connection.Settings, service, catalogue, serviceVersion string, revision int) (pluginmanager.Lock, error) {
	if err := validateService(service); err != nil {
		return pluginmanager.Lock{}, err
	}
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	catalogue, err = servicePluginCatalogue(cmd, settings, manager, service, catalogue)
	if err != nil {
		return pluginmanager.Lock{}, err
	}

	target, version, err := pluginServiceSyncTarget(cmd.Context(), settings, cmd, service, serviceVersion)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	release, err := manager.Resolve(cmd.Context(), catalogue, service, version, revision)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	if err := validateServiceManifest(cmd.Context(), cmd.Root(), service, release.Manifest); err != nil {
		return pluginmanager.Lock{}, err
	}
	return manager.InstallResolved(cmd.Context(), catalogue, target, release)
}

// List returns all prepared service locks.
func List(settings *connection.Settings, cmd *cobra.Command) ([]pluginmanager.Lock, error) {
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return nil, err
	}
	return manager.List()
}

// Show returns cached metadata for the selected service target.
func Show(settings *connection.Settings, cmd *cobra.Command, service string) (pluginmanager.Lock, error) {
	if err := validateService(service); err != nil {
		return pluginmanager.Lock{}, err
	}
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	return cachedServiceLock(settings, cmd, manager, service)
}

func validateService(service string) error {
	_, err := supportedPluginService(service)
	return err
}
