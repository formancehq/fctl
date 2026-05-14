package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	v3cmd "github.com/formancehq/fctl/v3/cmd"
)

const schemaVersion = 1

type inventory struct {
	SchemaVersion int             `json:"schemaVersion"`
	SourceModule  string          `json:"sourceModule"`
	RootUse       string          `json:"rootUse"`
	CommandCount  int             `json:"commandCount"`
	Commands      []commandRecord `json:"commands"`
}

type commandRecord struct {
	Path           string       `json:"path"`
	PathParts      []string     `json:"pathParts"`
	Use            string       `json:"use"`
	Aliases        []string     `json:"aliases,omitempty"`
	Short          string       `json:"short,omitempty"`
	Long           string       `json:"long,omitempty"`
	Hidden         bool         `json:"hidden,omitempty"`
	Deprecated     string       `json:"deprecated,omitempty"`
	Runnable       bool         `json:"runnable"`
	HasSubcommands bool         `json:"hasSubcommands"`
	Flags          []flagRecord `json:"flags,omitempty"`
}

type flagRecord struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Shorthand   string `json:"shorthand,omitempty"`
	Usage       string `json:"usage,omitempty"`
	Default     string `json:"default,omitempty"`
	Type        string `json:"type,omitempty"`
	NoOptDefVal string `json:"noOptDefVal,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Deprecated  string `json:"deprecated,omitempty"`
}

func main() {
	var output string
	flag.StringVar(&output, "output", "", "Path to write inventory JSON, or stdout when empty")
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		exitf("resolve home directory: %v", err)
	}

	root := v3cmd.NewRootCommand()
	root.DisableAutoGenTag = true

	records := collectCommands(root, homeDir)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Path < records[j].Path
	})

	doc := inventory{
		SchemaVersion: schemaVersion,
		SourceModule:  "github.com/formancehq/fctl/v3",
		RootUse:       root.Use,
		CommandCount:  len(records),
		Commands:      records,
	}

	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		exitf("marshal inventory: %v", err)
	}
	payload = append(payload, '\n')

	if output == "" || output == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			exitf("write stdout: %v", err)
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		exitf("create output directory: %v", err)
	}
	if err := os.WriteFile(output, payload, 0o644); err != nil {
		exitf("write %s: %v", output, err)
	}
}

func collectCommands(root *cobra.Command, homeDir string) []commandRecord {
	var records []commandRecord
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		records = append(records, commandRecord{
			Path:           cmd.CommandPath(),
			PathParts:      strings.Fields(cmd.CommandPath()),
			Use:            cmd.Use,
			Aliases:        sortedStrings(cmd.Aliases),
			Short:          cmd.Short,
			Long:           cmd.Long,
			Hidden:         cmd.Hidden,
			Deprecated:     cmd.Deprecated,
			Runnable:       cmd.Runnable(),
			HasSubcommands: cmd.HasSubCommands(),
			Flags:          collectFlags(cmd, homeDir),
		})

		children := cmd.Commands()
		sort.Slice(children, func(i, j int) bool {
			return children[i].Name() < children[j].Name()
		})
		for _, child := range children {
			walk(child)
		}
	}
	walk(root)
	return records
}

func collectFlags(cmd *cobra.Command, homeDir string) []flagRecord {
	var records []flagRecord
	records = append(records, flagsFromSet("local", cmd.LocalFlags(), homeDir)...)
	records = append(records, flagsFromSet("persistent", cmd.PersistentFlags(), homeDir)...)
	records = append(records, flagsFromSet("inherited", cmd.InheritedFlags(), homeDir)...)
	sort.Slice(records, func(i, j int) bool {
		if records[i].Name == records[j].Name {
			return records[i].Scope < records[j].Scope
		}
		return records[i].Name < records[j].Name
	})
	return records
}

func flagsFromSet(scope string, set *pflag.FlagSet, homeDir string) []flagRecord {
	if set == nil {
		return nil
	}

	var records []flagRecord
	set.VisitAll(func(f *pflag.Flag) {
		record := flagRecord{
			Name:        f.Name,
			Scope:       scope,
			Shorthand:   f.Shorthand,
			Usage:       f.Usage,
			Default:     normalizeDefault(f.DefValue, homeDir),
			NoOptDefVal: f.NoOptDefVal,
			Hidden:      f.Hidden,
			Deprecated:  f.Deprecated,
		}
		if f.Value != nil {
			record.Type = f.Value.Type()
		}
		records = append(records, record)
	})
	return records
}

func normalizeDefault(value string, homeDir string) string {
	if homeDir == "" {
		return value
	}
	if value == homeDir {
		return "$HOME"
	}
	return strings.ReplaceAll(value, homeDir+string(os.PathSeparator), "$HOME/")
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
