package plugin

import (
	"testing"
)

func TestFindCompatibleVersion(t *testing.T) {
	p := &InstalledPlugin{
		Name: "ledger",
		Versions: map[string]InstalledPluginVersion{
			"3.0.0": {CompatibleWith: ">= 3.0.0, < 3.3.0"},
			"3.3.0": {CompatibleWith: ">= 3.3.0, < 4.0.0"},
			"4.0.0": {CompatibleWith: ">= 4.0.0"},
		},
	}

	tests := []struct {
		serviceVersion string
		wantVersion    string
		wantFound      bool
	}{
		{"3.1.0", "3.0.0", true},
		{"3.0.0", "3.0.0", true},
		{"3.3.0", "3.3.0", true},
		{"3.5.2", "3.3.0", true},
		{"4.0.0", "4.0.0", true},
		{"4.2.1", "4.0.0", true},
		{"2.9.0", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.serviceVersion, func(t *testing.T) {
			got, found := p.FindCompatibleVersion(tt.serviceVersion)
			if found != tt.wantFound {
				t.Fatalf("FindCompatibleVersion(%q): found=%v, want %v", tt.serviceVersion, found, tt.wantFound)
			}
			if got != tt.wantVersion {
				t.Fatalf("FindCompatibleVersion(%q): got=%q, want %q", tt.serviceVersion, got, tt.wantVersion)
			}
		})
	}
}

func TestFindCompatibleVersionPicksHighest(t *testing.T) {
	p := &InstalledPlugin{
		Name: "ledger",
		Versions: map[string]InstalledPluginVersion{
			"3.0.0": {CompatibleWith: ">= 3.0.0"},
			"3.2.0": {CompatibleWith: ">= 3.0.0"},
			"3.5.0": {CompatibleWith: ">= 3.0.0"},
		},
	}

	got, found := p.FindCompatibleVersion("3.1.0")
	if !found {
		t.Fatal("expected to find a version")
	}
	if got != "3.5.0" {
		t.Fatalf("expected 3.5.0, got %q", got)
	}
}

func TestAddPluginVersion(t *testing.T) {
	cfg := &PluginsConfig{}

	cfg.AddPluginVersion("ledger", "3.0.0", InstalledPluginVersion{
		CompatibleWith: ">= 3.0.0",
	})

	p := cfg.FindInstalledPlugin("ledger")
	if p == nil {
		t.Fatal("expected plugin to be added")
	}
	if len(p.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(p.Versions))
	}

	cfg.AddPluginVersion("ledger", "3.3.0", InstalledPluginVersion{
		CompatibleWith: ">= 3.3.0",
	})

	if len(p.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(p.Versions))
	}
}

func TestRemovePluginVersion(t *testing.T) {
	cfg := &PluginsConfig{}
	cfg.AddPluginVersion("ledger", "3.0.0", InstalledPluginVersion{CompatibleWith: ">= 3.0.0"})
	cfg.AddPluginVersion("ledger", "3.3.0", InstalledPluginVersion{CompatibleWith: ">= 3.3.0"})

	cfg.RemovePluginVersion("ledger", "3.0.0")

	p := cfg.FindInstalledPlugin("ledger")
	if p == nil {
		t.Fatal("plugin should still exist")
	}
	if len(p.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(p.Versions))
	}

	cfg.RemovePluginVersion("ledger", "3.3.0")

	p = cfg.FindInstalledPlugin("ledger")
	if p != nil {
		t.Fatal("plugin should be removed when no versions left")
	}
}
