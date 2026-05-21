package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
)

// InstalledPluginVersion describes a single installed version of a plugin.
type InstalledPluginVersion struct {
	CompatibleWith string `json:"compatibleWith"`
	Path           string `json:"path,omitempty"`
}

// InstalledPlugin describes a plugin with potentially multiple installed versions.
type InstalledPlugin struct {
	Name     string                            `json:"name"`
	Versions map[string]InstalledPluginVersion  `json:"versions"`
}

// FindCompatibleVersion returns the highest installed version whose compatibleWith
// range satisfies the given service version. Returns empty string if none match.
func (p *InstalledPlugin) FindCompatibleVersion(serviceVersion string) (string, bool) {
	sv, err := semver.NewVersion(serviceVersion)
	if err != nil {
		return "", false
	}

	var bestVersion string
	var bestSemver *semver.Version

	for version, entry := range p.Versions {
		constraint, err := semver.NewConstraint(entry.CompatibleWith)
		if err != nil {
			continue
		}
		if !constraint.Check(sv) {
			continue
		}
		v, err := semver.NewVersion(version)
		if err != nil {
			continue
		}
		if bestSemver == nil || v.GreaterThan(bestSemver) {
			bestVersion = version
			bestSemver = v
		}
	}

	return bestVersion, bestVersion != ""
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

// AddPluginVersion adds or updates a specific version of a plugin.
func (c *PluginsConfig) AddPluginVersion(name, version string, entry InstalledPluginVersion) {
	p := c.FindInstalledPlugin(name)
	if p == nil {
		c.Plugins = append(c.Plugins, InstalledPlugin{
			Name:     name,
			Versions: map[string]InstalledPluginVersion{version: entry},
		})
		return
	}
	if p.Versions == nil {
		p.Versions = make(map[string]InstalledPluginVersion)
	}
	p.Versions[version] = entry
}

// RemovePlugin removes a plugin entry entirely (all versions).
func (c *PluginsConfig) RemovePlugin(name string) {
	for i := range c.Plugins {
		if c.Plugins[i].Name == name {
			c.Plugins = append(c.Plugins[:i], c.Plugins[i+1:]...)
			return
		}
	}
}

// RemovePluginVersion removes a specific version of a plugin.
func (c *PluginsConfig) RemovePluginVersion(name, version string) {
	p := c.FindInstalledPlugin(name)
	if p == nil {
		return
	}
	delete(p.Versions, version)
	if len(p.Versions) == 0 {
		c.RemovePlugin(name)
	}
}
