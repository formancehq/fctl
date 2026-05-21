package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
	"google.golang.org/protobuf/encoding/protojson"
)

// CachedManifest wraps a plugin manifest with cache metadata.
type CachedManifest struct {
	BinaryModTime int64  `json:"binaryModTime"`
	ManifestJSON  []byte `json:"manifest"`
}

// ManifestCachePath returns the path to the cached manifest for a plugin version.
func ManifestCachePath(configDir, name, version string) string {
	return filepath.Join(PluginsDir(configDir), name, version, "manifest.cache.json")
}

// LoadCachedManifest loads a cached manifest from disk. Returns nil if the cache
// is missing, corrupt, or stale (binary has been modified since caching).
func LoadCachedManifest(configDir, name, version string) (*pluginpb.PluginManifest, error) {
	cachePath := ManifestCachePath(configDir, name, version)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cached CachedManifest
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("corrupt manifest cache: %w", err)
	}

	// Check if binary has been modified since caching
	binaryPath := PluginBinaryPath(configDir, name, version)
	info, err := os.Stat(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat plugin binary: %w", err)
	}
	if info.ModTime().UnixNano() != cached.BinaryModTime {
		return nil, fmt.Errorf("manifest cache is stale")
	}

	var manifest pluginpb.PluginManifest
	if err := protojson.Unmarshal(cached.ManifestJSON, &manifest); err != nil {
		return nil, fmt.Errorf("invalid cached manifest: %w", err)
	}

	return &manifest, nil
}

// SaveCachedManifest writes a manifest to the cache, recording the current
// binary modification time for staleness detection.
func SaveCachedManifest(configDir, name, version string, manifest *pluginpb.PluginManifest) error {
	binaryPath := PluginBinaryPath(configDir, name, version)
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("cannot stat plugin binary: %w", err)
	}

	manifestJSON, err := protojson.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	cached := CachedManifest{
		BinaryModTime: info.ModTime().UnixNano(),
		ManifestJSON:  manifestJSON,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	cachePath := ManifestCachePath(configDir, name, version)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0o600)
}
