package cmd

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

func TestAuthDistributionCommandsPreserveLedgerDefaultsAndWrappers(t *testing.T) {
	root := &cobra.Command{Use: "fctl"}
	settings := &connection.Settings{}
	settings.Bind(root)
	settings.Directory = t.TempDir()
	if err := root.PersistentFlags().Set("auth-mode", "none"); err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentFlags().Set("ledger-url", "http://127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	for _, command := range []*cobra.Command{newPluginsInstallCommand(settings), newPluginsSyncCommand(settings), newPluginsShowCommand(settings)} {
		flag := command.Flags().Lookup("service")
		if flag == nil || flag.DefValue != "ledger" {
			t.Fatalf("%s changed Ledger default: %+v", command.Name(), flag)
		}
	}
	root.SetContext(t.Context())
	if err := installLedgerPlugin(root, settings, "", ""); err == nil || !strings.Contains(err.Error(), "ledger plugin executable") {
		t.Fatalf("Ledger install wrapper: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	root.SetContext(ctx)
	if err := syncLedgerPlugin(root, settings, filepath.Join(t.TempDir(), "catalogue.json"), "3.0.0", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ledger sync wrapper: %v", err)
	}
}
