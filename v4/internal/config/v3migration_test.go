package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/internal/credentials"
)

func TestLoadAndPlanV3Migration(t *testing.T) {
	dir := writeV3Fixture(t, true)

	state, err := LoadV3State(dir)
	if err != nil {
		t.Fatalf("load v3 state: %v", err)
	}
	plan, err := PlanV3Migration(state)
	if err != nil {
		t.Fatalf("plan migration: %v", err)
	}

	if plan.CurrentContext != "default" {
		t.Fatalf("expected current context default, got %q", plan.CurrentContext)
	}
	context := plan.Contexts["default"]
	if context.Kind != ContextKindCloudStack {
		t.Fatalf("expected cloud-stack context, got %q", context.Kind)
	}
	if context.CloudURL != "https://app.formance.cloud/api" ||
		context.Organization != "org_123" ||
		context.Stack != "stack_123" {
		t.Fatalf("unexpected migrated context: %#v", context)
	}
	if len(plan.CredentialMoves) != 1 {
		t.Fatalf("expected one credential move, got %d", len(plan.CredentialMoves))
	}
	if plan.CredentialMoves[0].Ref != context.Auth.TokenRef {
		t.Fatalf("expected auth token ref to match credential move")
	}
	if err := plan.Config().Validate(); err != nil {
		t.Fatalf("expected valid v4 config, got %v", err)
	}
}

func TestWriteMigrationStoresCredentials(t *testing.T) {
	dir := writeV3Fixture(t, true)
	state, err := LoadV3State(dir)
	if err != nil {
		t.Fatalf("load v3 state: %v", err)
	}
	plan, err := PlanV3Migration(state)
	if err != nil {
		t.Fatalf("plan migration: %v", err)
	}

	store := credentials.NewMemoryStore()
	output := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteMigration(context.Background(), output, plan, store); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	loaded, err := LoadFile(output)
	if err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
	if loaded.CurrentContext != "default" {
		t.Fatalf("expected default current context, got %q", loaded.CurrentContext)
	}

	value, err := store.Get(context.Background(), plan.CredentialMoves[0].Ref)
	if err != nil {
		t.Fatalf("get migrated credential: %v", err)
	}
	if value == "" {
		t.Fatalf("expected migrated credential value")
	}
}

func TestLoadV3StateExplainsMissingConfigDirectory(t *testing.T) {
	_, err := LoadV3State(t.TempDir())
	if err == nil {
		t.Fatal("expected missing v3 config error")
	}
	for _, expected := range []string{"read v3 config", "config.yml", "profiles/"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got %v", expected, err)
		}
	}
}

func writeV3Fixture(t *testing.T, withTokens bool) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "profiles", "default"), 0o700); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(`{"currentProfile":"default"}`), 0o600); err != nil {
		t.Fatalf("write v3 config: %v", err)
	}
	rootTokens := "null"
	if withTokens {
		rootTokens = `{"access":{"token":"access-token"},"id":{"token":"id-token"}}`
	}
	profile := `{
	  "membershipURI": "https://app.formance.cloud/api",
	  "rootTokens": ` + rootTokens + `,
	  "defaultOrganization": "org_123",
	  "defaultStack": "stack_123"
	}`
	if err := os.WriteFile(filepath.Join(dir, "profiles", "default", "profile.json"), []byte(profile), 0o600); err != nil {
		t.Fatalf("write v3 profile: %v", err)
	}
	return dir
}
