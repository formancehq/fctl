package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/formancehq/fctl-plugin-registry/main/registry.json"
)

// RegistrySchema is the top-level registry structure.
type RegistrySchema struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Plugins       map[string]RegistryPlugin `json:"plugins"`
}

// RegistryPlugin describes a single plugin in the registry.
type RegistryPlugin struct {
	Description string                            `json:"description"`
	Repo        string                            `json:"repo"`
	Latest      string                            `json:"latest"`
	Versions    map[string]RegistryPluginVersion   `json:"versions"`
}

// RegistryPluginVersion describes a specific version of a plugin.
type RegistryPluginVersion struct {
	MinCoreVersion string            `json:"minCoreVersion"`
	Binaries       map[string]string `json:"binaries"`
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

// FetchRegistry downloads and parses the registry JSON.
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
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, fmt.Errorf("failed to decode registry: %w", err)
	}
	return &reg, nil
}

// DownloadBinary downloads a plugin binary for the current OS/arch and saves it to disk.
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

// GetBinaryURL returns the download URL for the current platform from a version entry.
func GetBinaryURL(version RegistryPluginVersion) (string, error) {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	url, ok := version.Binaries[platform]
	if !ok {
		return "", fmt.Errorf("no binary available for platform %s", platform)
	}
	return url, nil
}
