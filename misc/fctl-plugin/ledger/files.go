package ledger

import (
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// File effects are declarative. SDK callers provide Body and consume Data;
// the host supplies local file/stdin reads and preserves historical exports.
func annotateFiles(manifest pluginsdk.Manifest) pluginsdk.Manifest {
	attachFiles(&manifest.Root, nil)
	annotateAliases(&manifest.Root, nil)
	return manifest
}

func attachFiles(command *pluginsdk.CommandSpec, parent []string) {
	path := append(append([]string{}, parent...), pluginsdk.CommandName(*command))
	switch strings.Join(path, " ") {
	case "ledger export":
		command.Files = &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}
	case "ledger import":
		command.Files = &pluginsdk.FileSpec{ReadArgument: new(1), ReadFlag: "file", ReadFormat: "string"}
	case "ledger transactions num":
		command.Files = &pluginsdk.FileSpec{ReadArgument: new(0), ReadFormat: "string"}
	case "ledger schemas insert":
		command.Files = &pluginsdk.FileSpec{ReadArgument: new(1), ReadFormat: "yaml"}
	case "ledger schemas get":
		command.Files = &pluginsdk.FileSpec{FormatFlag: "format", WriteFormat: "json"}
	}
	for i := range command.Subcommands {
		attachFiles(&command.Subcommands[i], path)
	}
}
