package plugins

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

func TestAuthDistributionCommandsPreserveLedgerDefaults(t *testing.T) {
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
	for _, command := range []*cobra.Command{newInstallCommand(settings), newSyncCommand(settings), newShowCommand(settings)} {
		flag := command.Flags().Lookup("service")
		if flag == nil || flag.DefValue != "ledger" {
			t.Fatalf("%s changed Ledger default: %+v", command.Name(), flag)
		}
	}
	root.AddCommand(NewCommand(settings))
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs([]string{"plugins", "install"})
	if err := root.ExecuteContext(t.Context()); err == nil || !strings.Contains(err.Error(), "ledger plugin executable") {
		t.Fatalf("Ledger install: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	root.SetArgs([]string{"plugins", "sync", "--catalogue", filepath.Join(t.TempDir(), "catalogue.json"), "--service-version", "3.0.0"})
	if err := root.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ledger sync: %v", err)
	}
}
