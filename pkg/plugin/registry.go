package plugin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/formancehq/poc-fctl-plugin-registry/main/registry.yaml"
)

// RegistrySchema is the top-level registry structure.
type RegistrySchema struct {
	Plugins map[string]RegistryPlugin `yaml:"plugins"`
}

// RegistryPlugin describes a single plugin in the registry.
type RegistryPlugin struct {
	Repo         string                            `yaml:"repo"`
	Type         string                            `yaml:"type"`
	Distribution string                            `yaml:"distribution"`
	Versions     map[string]RegistryPluginVersion  `yaml:"versions"`
}

// RegistryPluginVersion describes a specific version of a plugin.
type RegistryPluginVersion struct {
	CompatibleWith string `yaml:"compatibleWith"`
	Deprecated     string `yaml:"deprecated,omitempty"`
}

// FindBestVersion returns the highest plugin version whose compatibleWith range
// satisfies the given service version.
func (p *RegistryPlugin) FindBestVersion(serviceVersion string) (string, *RegistryPluginVersion, error) {
	sv, err := semver.NewVersion(serviceVersion)
	if err != nil {
		return "", nil, fmt.Errorf("invalid service version %q: %w", serviceVersion, err)
	}

	var bestVersion string
	var bestSemver *semver.Version
	var bestEntry RegistryPluginVersion

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
			bestEntry = entry
		}
	}

	if bestVersion == "" {
		return "", nil, fmt.Errorf("no compatible version found for service version %s", serviceVersion)
	}

	return bestVersion, &bestEntry, nil
}

// BinaryURL derives the download URL for a plugin binary from convention.
// Pattern: https://github.com/{repo}/releases/download/v{version}/fctl-plugin-{name}-{os}-{arch}
func (p *RegistryPlugin) BinaryURL(pluginName, version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/v%s/fctl-plugin-%s-%s-%s",
		p.Repo, version, pluginName, runtime.GOOS, runtime.GOARCH,
	)
}

// RegistryClient fetches plugin information from the remote registry.
type RegistryClient struct {
	URL        string
	HTTPClient *http.Client
}

// NewRegistryClient creates a new registry client with the default URL.
func NewRegistryClient(httpClient *http.Client) *RegistryClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RegistryClient{
		URL:        DefaultRegistryURL,
		HTTPClient: httpClient,
	}
}

// FetchRegistry downloads and parses the registry YAML.
func (r *RegistryClient) FetchRegistry() (*RegistrySchema, error) {
	resp, err := r.HTTPClient.Get(r.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var reg RegistrySchema
	if err := yaml.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, fmt.Errorf("failed to decode registry: %w", err)
	}
	return &reg, nil
}

// DownloadBinary downloads a plugin binary from the given URL and saves it to disk.
func (r *RegistryClient) DownloadBinary(binaryURL, destPath string) error {
	resp, err := r.HTTPClient.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create plugin file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write plugin file: %w", err)
	}

	return nil
}
