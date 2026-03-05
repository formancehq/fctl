package plugin

import (
	"context"
	"fmt"
	"os"

	"github.com/formancehq/go-libs/v3/logging"
)

// PluginManager discovers, loads, and manages the lifecycle of fctl plugins.
type PluginManager struct {
	configDir string
	loaded    []*LoadedPlugin
}

// NewPluginManager creates a new PluginManager.
func NewPluginManager(configDir string) *PluginManager {
	return &PluginManager{
		configDir: configDir,
	}
}

// DiscoverAndLoad finds all installed plugins and loads them.
func (pm *PluginManager) DiscoverAndLoad(ctx context.Context) {
	logger := logging.FromContext(ctx)

	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		logger.Debugf("Failed to load plugins config: %v", err)
		return
	}

	for _, p := range cfg.Plugins {
		binaryPath := p.Path
		if binaryPath == "" {
			binaryPath = PluginBinaryPath(pm.configDir, p.Name, p.Version)
		}

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			logger.Debugf("Plugin binary not found for %s at %s, skipping", p.Name, binaryPath)
			continue
		}

		loaded, err := LoadPlugin(p.Name, binaryPath)
		if err != nil {
			logger.Debugf("Failed to load plugin %s: %v", p.Name, err)
			continue
		}
		loaded.Version = p.Version
		pm.loaded = append(pm.loaded, loaded)
	}
}

// GetLoadedPlugins returns all successfully loaded plugins.
func (pm *PluginManager) GetLoadedPlugins() []*LoadedPlugin {
	return pm.loaded
}

// Shutdown kills all loaded plugin processes.
func (pm *PluginManager) Shutdown() {
	for _, p := range pm.loaded {
		p.Kill()
	}
	pm.loaded = nil
}

// InstallPlugin downloads and installs a plugin from the registry.
func (pm *PluginManager) InstallPlugin(name, version string, registry *RegistryClient) error {
	reg, err := registry.FetchRegistry()
	if err != nil {
		return err
	}

	pluginInfo, ok := reg.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found in registry", name)
	}

	if version == "" {
		version = pluginInfo.Latest
	}

	versionInfo, ok := pluginInfo.Versions[version]
	if !ok {
		return fmt.Errorf("version %s not found for plugin %q", version, name)
	}

	binaryURL, err := GetBinaryURL(versionInfo)
	if err != nil {
		return err
	}

	destPath := PluginBinaryPath(pm.configDir, name, version)
	if err := registry.DownloadBinary(binaryURL, destPath); err != nil {
		return err
	}

	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		return err
	}

	cfg.AddOrUpdatePlugin(InstalledPlugin{
		Name:    name,
		Version: version,
		Path:    destPath,
	})

	return SavePluginsConfig(pm.configDir, cfg)
}

// RemovePlugin uninstalls a plugin.
func (pm *PluginManager) RemovePlugin(name string) error {
	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		return err
	}

	installed := cfg.FindInstalledPlugin(name)
	if installed == nil {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	// Remove the binary
	binaryPath := installed.Path
	if binaryPath == "" {
		binaryPath = PluginBinaryPath(pm.configDir, name, installed.Version)
	}
	_ = os.RemoveAll(binaryPath)

	// Remove plugin directory
	pluginDir := PluginsDir(pm.configDir) + "/" + name
	_ = os.RemoveAll(pluginDir)

	cfg.RemovePlugin(name)
	return SavePluginsConfig(pm.configDir, cfg)
}
