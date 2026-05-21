package plugin

import (
	"testing"
)

func TestResolveUsesInstalledPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPluginManager(tmpDir, false)

	// Simulate a loaded plugin
	pm.loaded = append(pm.loaded, &LoadedPlugin{
		Name:           "ledger",
		Version:        "3.2.0",
		CompatibleWith: ">= 3.0.0",
	})

	// Save matching config so FindPluginForService works
	cfg := &PluginsConfig{}
	cfg.AddPluginVersion("ledger", "3.2.0", InstalledPluginVersion{
		CompatibleWith: ">= 3.0.0",
	})
	if err := SavePluginsConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	res, err := Resolve("ledger", "3.1.0", pm, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := res.(UsePlugin); !ok {
		t.Fatalf("expected UsePlugin, got %T", res)
	}
}

func TestResolveBuiltInFallback(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPluginManager(tmpDir, false)

	res, err := Resolve("ledger", "2.8.0", pm, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := res.(UseBuiltIn); !ok {
		t.Fatalf("expected UseBuiltIn, got %T", res)
	}
}

func TestResolvePluginTakesPrecedenceOverBuiltIn(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPluginManager(tmpDir, false)

	pm.loaded = append(pm.loaded, &LoadedPlugin{
		Name:           "ledger",
		Version:        "3.2.0",
		CompatibleWith: ">= 3.0.0",
	})

	cfg := &PluginsConfig{}
	cfg.AddPluginVersion("ledger", "3.2.0", InstalledPluginVersion{
		CompatibleWith: ">= 3.0.0",
	})
	if err := SavePluginsConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	// builtInCovers is true but plugin should still win
	res, err := Resolve("ledger", "3.1.0", pm, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := res.(UsePlugin); !ok {
		t.Fatalf("expected UsePlugin (takes precedence), got %T", res)
	}
}

func TestResolveFallsBackToBuiltInWhenNoPluginAndNoRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPluginManager(tmpDir, false)

	// No plugin, no registry, builtInCovers = false
	res, err := Resolve("ledger", "3.1.0", pm, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Should still return UseBuiltIn as final fallback
	if _, ok := res.(UseBuiltIn); !ok {
		t.Fatalf("expected UseBuiltIn as fallback, got %T", res)
	}
}
