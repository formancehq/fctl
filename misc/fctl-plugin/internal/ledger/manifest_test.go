package ledger

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

//nolint:gocognit // This recursive inventory checks the complete command and selector contract.
func TestManifestOfflineAndEveryHistoricalCommand(t *testing.T) {
	m, err := New(nil).GetManifest(t.Context())
	if err != nil || m.Name != "ledger" || m.Service != "ledger" || m.Version != "1.0.0" || m.Root.Use != "ledger" || m.ProtocolVersion != pluginsdk.ProtocolVersion {
		t.Fatalf("manifest %+v: %v", m, err)
	}
	want := strings.Fields("create send stats server-infos list set-metadata delete-metadata export import")
	want = append(want, "accounts list", "accounts show", "accounts set-metadata", "accounts delete-metadata", "transactions list", "transactions num", "transactions revert", "transactions show", "transactions set-metadata", "transactions delete-metadata", "schemas insert", "schemas get", "schemas list", "volumes list")
	var visit func(pluginsdk.CommandSpec)
	count := 0
	visit = func(c pluginsdk.CommandSpec) {
		if c.Runnable {
			count++
			for _, input := range c.Inputs {
				if input.Source == nil {
					continue
				}
				source, err := pluginsdk.FindCommand(m, input.Source.CommandPath)
				if err != nil || !source.Runnable {
					t.Errorf("invalid selector source %v: %v", input.Source.CommandPath, err)
				}
			}
		}
		for _, child := range c.Subcommands {
			visit(child)
		}
	}
	visit(m.Root)
	if count != len(want) {
		t.Errorf("got %d executable commands, want %d", count, len(want))
	}
	for _, path := range want {
		c, err := pluginsdk.FindCommand(m, append([]string{"ledger"}, strings.Fields(path)...))
		if err != nil || !c.Runnable {
			t.Errorf("missing historical command %s: %v", path, err)
		}
	}
	m.Root.Subcommands[0].Use = "corrupted"
	fresh, err := New(nil).GetManifest(t.Context())
	if err != nil || fresh.Root.Subcommands[0].Use != "create <name>" {
		t.Fatalf("manifest was shared: %v", err)
	}
}
