package plugin_test

import (
	"context"
	"io"
	"net/http"
	"slices"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
)

func aliasManifest() pluginsdk.Manifest {
	manifest := testManifest()
	manifest.Root.Aliases = []string{"legacy_ledger"}
	manifest.Root.Subcommands[0].Use = "bank-accounts NAME"
	manifest.Root.Subcommands[0].Aliases = []string{"bank_accounts", "old-accounts"}
	return manifest
}

func TestRegistryRejectsInvalidAliases(t *testing.T) {
	t.Parallel()
	for _, alias := range []string{"", " ", "bank accounts", " bank_accounts", "bank_accounts\n", "BANK_ACCOUNTS", "bank/accounts", "--bank-accounts", "help", "bank-accounts"} {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()
			manifest := aliasManifest()
			manifest.Root.Subcommands[0].Aliases = []string{alias}
			assertInvalidAliasManifest(t, manifest)
		})
	}
	for _, alias := range []string{"help", "completion", "version", "profiles", "login", "logout", "ledger"} {
		manifest := aliasManifest()
		manifest.Root.Aliases = []string{alias}
		assertInvalidAliasManifest(t, manifest)
	}
	manifest := aliasManifest()
	manifest.Root.Subcommands[0].Use = "bank_accounts NAME"
	assertInvalidAliasManifest(t, manifest)
}

func assertInvalidAliasManifest(t *testing.T, manifest pluginsdk.Manifest) {
	t.Helper()
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, inertFactory); err == nil {
		t.Fatal("invalid alias metadata accepted")
	}
}

func TestRegistryRejectsSiblingAliasConflicts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		edit func(*pluginsdk.Manifest)
	}{
		{"duplicate leaf alias", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Aliases = []string{"old", "old"} }},
		{"duplicate root alias", func(m *pluginsdk.Manifest) { m.Root.Aliases = []string{"old", "old"} }},
		{"alias to canonical", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Aliases = []string{"list"} }},
		{"canonical to alias", func(m *pluginsdk.Manifest) { m.Root.Subcommands[1].Aliases = []string{"bank-accounts"} }},
		{"alias to alias", func(m *pluginsdk.Manifest) { m.Root.Subcommands[1].Aliases = []string{"bank_accounts"} }},
		{"historical h collision", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Aliases = []string{"h"}
			m.Root.Subcommands[1].Aliases = []string{"h"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := aliasManifest()
			manifest.Root.Subcommands = append(manifest.Root.Subcommands, pluginsdk.CommandSpec{Use: "list", Runnable: true})
			tc.edit(&manifest)
			assertInvalidAliasManifest(t, manifest)
		})
	}
}

func TestRegistryRejectsRootAliasConflicts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		root, alias   string
		existingAlias string
	}{
		{"alias to root", "payments", "ledger", "legacy_ledger"},
		{"root to alias", "payments", "old_payments", "payments"},
		{"alias to alias", "payments", "legacy_ledger", "legacy_ledger"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			registry := &plugin.Registry{}
			existing := aliasManifest()
			existing.Root.Aliases = []string{tc.existingAlias}
			if err := registry.Register(t.Context(), &fakePlugin{manifest: existing}, inertFactory); err != nil {
				t.Fatal(err)
			}
			manifest := aliasManifest()
			manifest.Name, manifest.Root.Use, manifest.Root.Aliases = tc.root, tc.root, []string{tc.alias}
			if err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, inertFactory); err == nil {
				t.Fatal("root alias conflict accepted")
			}
			if len(registry.List()) != 1 {
				t.Fatal("failed registration modified the registry")
			}
		})
	}
}

func TestRegistryAliasesAreIsolated(t *testing.T) {
	t.Parallel()
	manifest := aliasManifest()
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, inertFactory); err != nil {
		t.Fatal(err)
	}
	manifest.Root.Aliases[0] = "provider_change"
	manifest.Root.Subcommands[0].Aliases[0] = "provider_child_change"
	listed := registry.List()
	listed[0].Root.Aliases[0] = "list_change"
	listed[0].Root.Subcommands[0].Aliases[0] = "list_child_change"
	again := registry.List()[0]
	if again.Root.Aliases[0] != "legacy_ledger" || again.Root.Subcommands[0].Aliases[0] != "bank_accounts" {
		t.Fatal("alias slices share provider or list storage")
	}
}

func TestAdapterAliasRoutesRemainCanonical(t *testing.T) {
	t.Parallel()
	called := false
	root := attach(t, aliasManifest(), func(context.Context, string) (*api.Client, error) {
		return api.New("https://ledger.example", &http.Client{})
	}, func(*http.Client) pluginsdk.Plugin {
		return &fakePlugin{execute: func(_ context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			called = true
			if !slices.Equal(request.CommandPath, []string{"ledger", "bank-accounts"}) || !request.ChangedFlags["confirm"] || request.ChangedFlags["limit"] {
				t.Fatalf("alias invocation changed canonical routing or flags: %+v", request)
			}
			return pluginsdk.ExecuteResponse{}, nil
		}}
	})
	root.SetArgs([]string{"legacy_ledger", "bank_accounts", "fixture", "--tenant=org", "--confirm"})
	root.SetOut(io.Discard)
	if err := root.ExecuteContext(t.Context()); err != nil || !called {
		t.Fatalf("alias command did not execute: %v", err)
	}
}

func TestHistoricalHAliasPreservesHelpFlag(t *testing.T) {
	t.Parallel()
	manifest := aliasManifest()
	manifest.Root.Subcommands[0].Aliases = []string{"h"}
	called := false
	root := attach(t, manifest, func(context.Context, string) (*api.Client, error) {
		called = true
		return api.New("https://ledger.example", &http.Client{})
	}, inertFactory)
	root.SetOut(io.Discard)
	root.SetArgs([]string{"ledger", "h", "-h"})
	if err := root.ExecuteContext(t.Context()); err != nil || called {
		t.Fatalf("historical h alias did not preserve -h: %v", err)
	}
	request := pluginsdk.ExecuteRequest{CommandPath: []string{"ledger", "h"}, Args: []string{"fixture"}, Flags: map[string]string{"tenant": "org", "confirm": "true"}}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil || !slices.Equal(normalized.CommandPath, []string{"ledger", "bank-accounts"}) {
		t.Fatalf("historical h alias normalization = %v (%v)", normalized.CommandPath, err)
	}
}
