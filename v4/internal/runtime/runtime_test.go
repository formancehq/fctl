package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

func TestNewResolvesCurrentStackContext(t *testing.T) {
	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth:     config.Auth{Method: config.AuthMethodNone},
				API:      map[string]string{"ledger": string(config.APIPolicyPinned)},
			},
		},
	})

	rt, err := New(context.Background(), Options{
		ConfigPath:      configPath,
		Credentials:     credentials.NewMemoryStore(),
		Manifest:        capabilities.Manifest{SpecVersion: "test"},
		Compatibility:   capabilities.ComponentCompatibility{},
		ContextOverride: config.ContextOverride{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if rt.ContextName != "local" {
		t.Fatalf("expected local context, got %q", rt.ContextName)
	}
	if rt.Target.Kind != TargetKindStack || rt.Target.URL != "http://localhost/api" {
		t.Fatalf("unexpected target: %#v", rt.Target)
	}
	if policy := rt.APIPolicyFor("ledger"); policy != config.APIPolicyPinned {
		t.Fatalf("expected pinned policy, got %q", policy)
	}
	if policy := rt.APIPolicyFor("payments"); policy != config.APIPolicyLatestCompatible {
		t.Fatalf("expected default latest-compatible policy, got %q", policy)
	}
}

func TestNewUsesContextOverride(t *testing.T) {
	configPath := writeRuntimeConfig(t, config.Config{
		Version:        config.Version,
		CurrentContext: "local",
		Contexts: map[string]config.Context{
			"local": {
				Kind:     config.ContextKindStack,
				StackURL: "http://localhost/api",
				Auth:     config.Auth{Method: config.AuthMethodNone},
			},
			"cloud": {
				Kind:     config.ContextKindCloud,
				CloudURL: "https://app.formance.cloud/api",
				Auth:     config.Auth{Method: config.AuthMethodCloudDevice},
			},
		},
	})

	rt, err := New(context.Background(), Options{
		ConfigPath:      configPath,
		ContextOverride: config.ContextOverride{Name: "cloud"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rt.ContextName != "cloud" {
		t.Fatalf("expected cloud context, got %q", rt.ContextName)
	}
	if rt.Target.Kind != TargetKindCloud || rt.Target.URL != "https://app.formance.cloud/api" {
		t.Fatalf("unexpected target: %#v", rt.Target)
	}
}

func TestNewRequiresConfigPath(t *testing.T) {
	_, err := New(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected config path error")
	}
}

func writeRuntimeConfig(t *testing.T, cfg config.Config) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.SaveFile(path, cfg); err != nil {
		t.Fatalf("save runtime config: %v", err)
	}
	return path
}
