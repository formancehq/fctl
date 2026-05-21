package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
)

func TestCacheRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	name := "ledger"
	version := "3.2.0"

	// Create a fake binary so stat works
	binaryDir := filepath.Join(PluginsDir(tmpDir), name, version)
	if err := os.MkdirAll(binaryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := PluginBinaryPath(tmpDir, name, version)
	if err := os.WriteFile(binaryPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := &pluginpb.PluginManifest{
		Name:        "ledger",
		Version:     "3.2.0",
		Description: "Ledger v3 commands",
		RootCommand: &pluginpb.CommandSpec{
			Use:   "ledger",
			Short: "Ledger management",
		},
	}

	// Save
	if err := SaveCachedManifest(tmpDir, name, version, manifest); err != nil {
		t.Fatalf("SaveCachedManifest: %v", err)
	}

	// Load
	loaded, err := LoadCachedManifest(tmpDir, name, version)
	if err != nil {
		t.Fatalf("LoadCachedManifest: %v", err)
	}

	if loaded.Name != manifest.Name {
		t.Fatalf("Name mismatch: %q != %q", loaded.Name, manifest.Name)
	}
	if loaded.Version != manifest.Version {
		t.Fatalf("Version mismatch: %q != %q", loaded.Version, manifest.Version)
	}
	if loaded.RootCommand == nil || loaded.RootCommand.Use != "ledger" {
		t.Fatal("RootCommand not preserved")
	}
}

func TestCacheStaleAfterBinaryChange(t *testing.T) {
	tmpDir := t.TempDir()
	name := "ledger"
	version := "3.2.0"

	binaryDir := filepath.Join(PluginsDir(tmpDir), name, version)
	if err := os.MkdirAll(binaryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := PluginBinaryPath(tmpDir, name, version)
	if err := os.WriteFile(binaryPath, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := &pluginpb.PluginManifest{Name: "ledger", Version: "3.2.0"}
	if err := SaveCachedManifest(tmpDir, name, version, manifest); err != nil {
		t.Fatal(err)
	}

	// Modify the binary
	if err := os.WriteFile(binaryPath, []byte("v2-modified"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Cache should be stale
	_, err := LoadCachedManifest(tmpDir, name, version)
	if err == nil {
		t.Fatal("expected stale cache error after binary modification")
	}
}
