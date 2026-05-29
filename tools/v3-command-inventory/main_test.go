package main

import (
	"os"
	"strings"
	"testing"

	v3cmd "github.com/formancehq/fctl/v3/cmd"
)

func TestCollectCommandsIncludesImportantV3Paths(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home directory: %v", err)
	}

	records := collectCommands(v3cmd.NewRootCommand(), homeDir)
	byPath := map[string]commandRecord{}
	for _, record := range records {
		byPath[record.Path] = record
	}

	for _, path := range []string{
		"fctl ledger transactions list",
		"fctl payments connectors update-config stripe",
		"fctl orchestration workflows create",
		"fctl search",
		"fctl stack create",
	} {
		if _, ok := byPath[path]; !ok {
			t.Fatalf("expected inventory to include %q", path)
		}
	}

	if !byPath["fctl orchestration"].Hidden {
		t.Fatalf("expected fctl orchestration to be marked hidden")
	}
}

func TestCollectCommandsNormalizesHomeDirectoryDefaults(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home directory: %v", err)
	}

	records := collectCommands(v3cmd.NewRootCommand(), homeDir)
	for _, command := range records {
		for _, flag := range command.Flags {
			if homeDir != "" && strings.Contains(flag.Default, homeDir) {
				t.Fatalf("flag %s on %s leaks home directory in default %q", flag.Name, command.Path, flag.Default)
			}
		}
	}
}
