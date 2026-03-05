package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// InstalledPlugin describes a plugin entry in plugins.json.
type InstalledPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// PluginsConfig is the top-level structure for ~/.config/formance/fctl/plugins.json.
type PluginsConfig struct {
	Plugins []InstalledPlugin `json:"plugins"`
}

// PluginsDir returns the base directory for plugin binaries.
func PluginsDir(configDir string) string {
	return filepath.Join(configDir, "plugins")
}

// PluginBinaryPath returns the full path to a plugin binary.
func PluginBinaryPath(configDir, name, version string) string {
	return filepath.Join(PluginsDir(configDir), name, version, "fctl-plugin-"+name)
}

// PluginsConfigPath returns the path to plugins.json.
func PluginsConfigPath(configDir string) string {
	return filepath.Join(configDir, "plugins.json")
}

// LoadPluginsConfig reads the plugins.json configuration file.
func LoadPluginsConfig(configDir string) (*PluginsConfig, error) {
	path := PluginsConfigPath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PluginsConfig{}, nil
		}
		return nil, err
	}

	cfg := &PluginsConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SavePluginsConfig writes the plugins.json configuration file.
func SavePluginsConfig(configDir string, cfg *PluginsConfig) error {
	path := PluginsConfigPath(configDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// FindInstalledPlugin returns the installed plugin entry by name, or nil.
func (c *PluginsConfig) FindInstalledPlugin(name string) *InstalledPlugin {
	for i := range c.Plugins {
		if c.Plugins[i].Name == name {
			return &c.Plugins[i]
		}
	}
	return nil
}

// AddOrUpdatePlugin adds or updates a plugin entry.
func (c *PluginsConfig) AddOrUpdatePlugin(p InstalledPlugin) {
	for i := range c.Plugins {
		if c.Plugins[i].Name == p.Name {
			c.Plugins[i] = p
			return
		}
	}
	c.Plugins = append(c.Plugins, p)
}

// RemovePlugin removes a plugin entry by name.
func (c *PluginsConfig) RemovePlugin(name string) {
	for i := range c.Plugins {
		if c.Plugins[i].Name == name {
			c.Plugins = append(c.Plugins[:i], c.Plugins[i+1:]...)
			return
		}
	}
}
