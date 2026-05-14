package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		Version:        Version,
		CurrentContext: "local",
		Contexts: map[string]Context{
			"local": {
				Kind:     ContextKindStack,
				StackURL: "http://localhost/api",
				Auth: Auth{
					Method:    AuthMethodClientCredentials,
					IssuerURL: "http://localhost/api/auth",
					ClientID:  "testing",
					SecretRef: "keyring://formance/local/testing",
				},
				Defaults: map[string]string{"ledger": "default"},
				API:      map[string]string{"ledger": string(APIPolicyLatestCompatible)},
			},
		},
	}
}

func TestValidateAcceptsStackContext(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateRejectsMissingCurrentContext(t *testing.T) {
	cfg := validConfig()
	cfg.CurrentContext = "missing"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), `current context "missing" does not exist`) {
		t.Fatalf("expected missing current context error, got %v", err)
	}
}

func TestValidateRejectsSecretInlineByRequiringReference(t *testing.T) {
	cfg := validConfig()
	cfg.Contexts["local"] = Context{
		Kind:     ContextKindStack,
		StackURL: "http://localhost/api",
		Auth: Auth{
			Method:    AuthMethodClientCredentials,
			IssuerURL: "http://localhost/api/auth",
			ClientID:  "testing",
		},
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "secretRef is required") {
		t.Fatalf("expected secretRef validation error, got %v", err)
	}
}

func TestResolveCurrentContextUsesOverride(t *testing.T) {
	cfg := validConfig()
	cfg.Contexts["other"] = Context{
		Kind:     ContextKindStack,
		StackURL: "http://other/api",
		Auth:     Auth{Method: AuthMethodNone},
	}

	name, context, err := ResolveCurrentContext(cfg, ContextOverride{Name: "other"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "other" || context.StackURL != "http://other/api" {
		t.Fatalf("unexpected context resolution: %s %#v", name, context)
	}
}

func TestResolveCurrentContextUsesSingleContextWhenCurrentUnset(t *testing.T) {
	cfg := validConfig()
	cfg.CurrentContext = ""

	name, _, err := ResolveCurrentContext(cfg, ContextOverride{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "local" {
		t.Fatalf("expected local context, got %q", name)
	}
}

func TestContextOverrideFromEnv(t *testing.T) {
	override := ContextOverrideFromEnv(func(key string) string {
		if key != EnvContext {
			t.Fatalf("unexpected env key %q", key)
		}
		return "local"
	})
	if override.Name != "local" {
		t.Fatalf("expected local override, got %q", override.Name)
	}
}

func TestLoadSaveFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	cfg := validConfig()

	if err := SaveFile(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected config mode 0600, got %o", mode)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.CurrentContext != cfg.CurrentContext {
		t.Fatalf("expected current context %q, got %q", cfg.CurrentContext, loaded.CurrentContext)
	}
	if loaded.Contexts["local"].Auth.SecretRef != cfg.Contexts["local"].Auth.SecretRef {
		t.Fatalf("expected secret ref to round trip")
	}
}
