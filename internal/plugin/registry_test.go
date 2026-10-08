package plugin_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

func inertFactory(*http.Client) pluginsdk.Plugin { return &fakePlugin{} }

func TestRegistryRejectsInvalidManifests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*pluginsdk.Manifest)
	}{
		{"protocol", func(m *pluginsdk.Manifest) { m.ProtocolVersion = 2 }},
		{"empty name", func(m *pluginsdk.Manifest) { m.Name = "" }},
		{"empty version", func(m *pluginsdk.Manifest) { m.Version = "" }},
		{"empty service", func(m *pluginsdk.Manifest) { m.Service = "" }},
		{"mismatched root", func(m *pluginsdk.Manifest) { m.Root.Use = "auth" }},
		{"reserved root", func(m *pluginsdk.Manifest) { m.Name = "login"; m.Root.Use = "login" }},
		{"leading command whitespace", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Use = " write" }},
		{"command tab", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Use = "write\tNAME" }},
		{"empty command", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Use = "" }},
		{"invalid command", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Use = "--bad" }},
		{"reserved help", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Use = "help" }},
		{"negative args", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Args.Min = -1 }},
		{"inverted args", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Args.Max = 0 }},
		{"group args", func(m *pluginsdk.Manifest) { m.Root.Args = pluginsdk.ArgsSpec{Min: 1, Max: 1} }},
		{"group confirm", func(m *pluginsdk.Manifest) { m.Root.Confirm = true }},
		{"duplicate command", func(m *pluginsdk.Manifest) { m.Root.Subcommands = append(m.Root.Subcommands, m.Root.Subcommands[0]) }},
		{"duplicate flag", func(m *pluginsdk.Manifest) { m.Root.Flags = append(m.Root.Flags, m.Root.Flags[0]) }},
		{"inherited flag collision", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Flags = append(m.Root.Subcommands[0].Flags, m.Root.Flags[0])
		}},
		{"reserved flag", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Name = "connection" }},
		{"unsupported type", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Type = "duration" }},
		{"bool default", func(m *pluginsdk.Manifest) { m.Root.Flags[1].Default = "1" }},
		{"uint32 default", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags[2].Default = "4294967296" }},
		{"negative uint32", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags[2].Default = "-1" }},
		{"reserved shorthand", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Shorthand = "o" }},
		{"long shorthand", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Shorthand = "ab" }},
		{"invalid shorthand", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Shorthand = "-" }},
		{"duplicate shorthand", func(m *pluginsdk.Manifest) { m.Root.Flags[0].Shorthand = "x"; m.Root.Flags[1].Shorthand = "x" }},
		{"inherited shorthand collision", func(m *pluginsdk.Manifest) {
			m.Root.Flags[0].Shorthand = "x"
			m.Root.Subcommands[0].Flags[0].Shorthand = "x"
		}},
		{"non-string body", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags[2].Body = true }},
		{"duplicate body", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Flags = append(m.Root.Subcommands[0].Flags, pluginsdk.FlagSpec{Name: "other", Type: "string", Body: true})
		}},
		{"missing confirm flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags = m.Root.Subcommands[0].Flags[1:] }},
		{"confirm true by default", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags[0].Default = "true" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := testManifest()
			tc.mutate(&m)
			var registry plugin.Registry
			if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, inertFactory); err == nil {
				t.Fatal("invalid manifest accepted")
			}
			if len(registry.List()) != 0 {
				t.Fatal("invalid manifest entered registry")
			}
		})
	}
}

func TestRegistryFreezeAndIndependentMetadata(t *testing.T) {
	t.Parallel()
	var registry plugin.Registry
	m := testManifest()
	if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, inertFactory); err != nil {
		t.Fatal(err)
	}
	m.Root.Flags[0].Name = "mutated"
	first := registry.List()
	first[0].Root.Subcommands[0].Flags[0].Name = "mutated"
	if got := registry.List(); !reflect.DeepEqual(got, []pluginsdk.Manifest{testManifest()}) {
		t.Fatal("registry metadata aliases callers")
	}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: testManifest()}, inertFactory); err == nil {
		t.Fatal("duplicate root accepted")
	}
	registry.Freeze()
	other := testManifest()
	other.Name = "auth"
	other.Root.Use = "auth"
	if err := registry.Register(t.Context(), &fakePlugin{manifest: other}, inertFactory); err == nil {
		t.Fatal("frozen registry accepted plugin")
	}
}

func TestRegistryMetadataFailures(t *testing.T) {
	t.Parallel()
	var registry plugin.Registry
	sentinel := errors.New("manifest unavailable")
	if err := registry.Register(t.Context(), &fakePlugin{manifestErr: sentinel}, inertFactory); !errors.Is(err, sentinel) {
		t.Fatalf("manifest error lost: %v", err)
	}
	if err := registry.Register(t.Context(), nil, inertFactory); err == nil {
		t.Fatal("nil metadata accepted")
	}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: testManifest()}, nil); err == nil {
		t.Fatal("nil factory accepted")
	}
}

func TestAdapterHostCollisionsAndAtomicAttachment(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"command", "alias", "flag", "shorthand"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root := &cobra.Command{Use: "fctl"}
			m := testManifest()
			configureHostCollision(root, &m, kind)
			var registry plugin.Registry
			first := testManifest()
			first.Name = "auth"
			first.Root.Use = "auth"
			first.Root.Flags = nil
			first.Root.Subcommands = nil
			if err := registry.Register(t.Context(), &fakePlugin{manifest: first}, inertFactory); err != nil {
				t.Fatal(err)
			}
			if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, inertFactory); err != nil {
				t.Fatal(err)
			}
			before := len(root.Commands())
			adapter := plugin.NewCommand(&registry, func(context.Context, string) (*api.Client, error) {
				t.Error("attachment resolved HTTP")
				return nil, nil
			})
			if err := adapter.AddTo(root); err == nil {
				t.Fatal("host collision accepted")
			}
			if len(root.Commands()) != before {
				t.Fatal("failed attachment partially changed root")
			}
		})
	}
}

func configureHostCollision(root *cobra.Command, m *pluginsdk.Manifest, kind string) {
	switch kind {
	case "command":
		root.AddCommand(&cobra.Command{Use: "ledger"})
	case "alias":
		root.AddCommand(&cobra.Command{Use: "existing", Aliases: []string{"ledger"}})
	case "flag":
		root.PersistentFlags().String("tenant", "", "host")
	case "shorthand":
		root.PersistentFlags().StringP("host", "x", "", "host")
		m.Root.Flags[0].Shorthand = "x"
	}
}
