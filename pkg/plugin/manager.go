package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/formancehq/go-libs/v3/logging"
)

// PluginManager discovers, loads, and manages the lifecycle of fctl plugins.
type PluginManager struct {
	configDir string
	debug     bool
	loaded    []*LoadedPlugin
}

// NewPluginManager creates a new PluginManager.
func NewPluginManager(configDir string, debug bool) *PluginManager {
	return &PluginManager{
		configDir: configDir,
		debug:     debug,
	}
}

// DiscoverAndLoad finds all installed plugins and prepares them for use.
// Plugins with a cached manifest are loaded lazily (no process spawned).
// Plugins without a cache are spawned to fetch their manifest, then cached.
func (pm *PluginManager) DiscoverAndLoad(ctx context.Context) {
	logger := logging.FromContext(ctx)

	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		logger.Debugf("Failed to load plugins config: %v", err)
		return
	}

	for _, p := range cfg.Plugins {
		for version, entry := range p.Versions {
			binaryPath := entry.Path
			if binaryPath == "" {
				binaryPath = PluginBinaryPath(pm.configDir, p.Name, version)
			}

			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				logger.Debugf("Plugin binary not found for %s@%s at %s, skipping", p.Name, version, binaryPath)
				continue
			}

			// Try cached manifest first (no process spawn)
			if manifest, err := LoadCachedManifest(pm.configDir, p.Name, version); err == nil {
				lazy := NewLazyPluginClient(p.Name, binaryPath, manifest, pm.debug)
				pm.loaded = append(pm.loaded, &LoadedPlugin{
					Name:           p.Name,
					Version:        version,
					CompatibleWith: entry.CompatibleWith,
					Client:         lazy,
				})
				logger.Debugf("Plugin %s@%s loaded from cache", p.Name, version)
				continue
			}

			// Cache miss: spawn plugin, get manifest, cache it
			loaded, err := LoadPlugin(p.Name, binaryPath)
			if err != nil {
				logger.Debugf("Failed to load plugin %s@%s: %v", p.Name, version, err)
				continue
			}
			loaded.Version = version
			loaded.CompatibleWith = entry.CompatibleWith

			manifest, err := loaded.Client.GetManifest(ctx)
			if err != nil {
				logger.Debugf("Failed to get manifest for %s@%s: %v", p.Name, version, err)
				loaded.Kill()
				continue
			}

			if err := SaveCachedManifest(pm.configDir, p.Name, version, manifest); err != nil {
				logger.Debugf("Failed to cache manifest for %s@%s: %v", p.Name, version, err)
			}

			pm.loaded = append(pm.loaded, loaded)
			logger.Debugf("Plugin %s@%s loaded and cached", p.Name, version)
		}
	}
}

// GetLoadedPlugins returns all successfully loaded plugins.
func (pm *PluginManager) GetLoadedPlugins() []*LoadedPlugin {
	return pm.loaded
}

// FindPluginForService returns the best loaded plugin for a given service version.
func (pm *PluginManager) FindPluginForService(serviceName, serviceVersion string) *LoadedPlugin {
	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		return nil
	}
	p := cfg.FindInstalledPlugin(serviceName)
	if p == nil {
		return nil
	}
	bestVersion, found := p.FindCompatibleVersion(serviceVersion)
	if !found {
		return nil
	}
	for _, loaded := range pm.loaded {
		if loaded.Name == serviceName && loaded.Version == bestVersion {
			return loaded
		}
	}
	return nil
}

// Shutdown kills all loaded plugin processes.
func (pm *PluginManager) Shutdown() {
	for _, p := range pm.loaded {
		p.Kill()
	}
	pm.loaded = nil
}

// InstallFromRegistry downloads and installs a plugin version from the registry.
func (pm *PluginManager) InstallFromRegistry(name, version string, registryPlugin RegistryPlugin, registry *RegistryClient) error {
	versionInfo, ok := registryPlugin.Versions[version]
	if !ok {
		return fmt.Errorf("version %s not found for plugin %q", version, name)
	}

	binaryURL := registryPlugin.BinaryURL(name, version)
	destPath := PluginBinaryPath(pm.configDir, name, version)

	if err := registry.DownloadBinary(binaryURL, destPath); err != nil {
		return err
	}

	// Spawn to fetch and cache the manifest
	loaded, err := LoadPlugin(name, destPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin after install: %w", err)
	}
	defer loaded.Kill()

	ctx := context.Background()
	manifest, err := loaded.Client.GetManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get manifest after install: %w", err)
	}

	if err := SaveCachedManifest(pm.configDir, name, version, manifest); err != nil {
		return fmt.Errorf("failed to cache manifest: %w", err)
	}

	// Save config
	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		return err
	}

	cfg.AddPluginVersion(name, version, InstalledPluginVersion{
		CompatibleWith: versionInfo.CompatibleWith,
	})

	return SavePluginsConfig(pm.configDir, cfg)
}

// RemovePlugin uninstalls a plugin (all versions).
func (pm *PluginManager) RemovePlugin(name string) error {
	cfg, err := LoadPluginsConfig(pm.configDir)
	if err != nil {
		return err
	}

	installed := cfg.FindInstalledPlugin(name)
	if installed == nil {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	// Remove the entire plugin directory (all versions)
	pluginDir := filepath.Join(PluginsDir(pm.configDir), name)
	_ = os.RemoveAll(pluginDir)

	cfg.RemovePlugin(name)
	return SavePluginsConfig(pm.configDir, cfg)
}
