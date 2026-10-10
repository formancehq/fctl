package auth

import (
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

var historicalAliases = map[string][]string{
	"auth clients": {"client", "c"}, "auth clients create": {"c"}, "auth clients delete": {"d", "del"}, "auth clients list": {"ls", "l"}, "auth clients show": {"s"}, "auth clients update": {"u", "upd"},
	"auth clients secrets": {"sec"}, "auth clients secrets create": {"c"}, "auth clients secrets delete": {"d"}, "auth users": {"u", "user"}, "auth users list": {"ls", "l"}, "auth users show": {"s"}, "auth clients users list": {"ls", "l"},
}

func annotateAliases(manifest pluginsdk.Manifest) pluginsdk.Manifest {
	attachAliases(&manifest.Root, nil)
	return manifest
}
func attachAliases(command *pluginsdk.CommandSpec, path []string) {
	path = append(append([]string{}, path...), pluginsdk.CommandName(*command))
	command.Aliases = append([]string{}, historicalAliases[strings.Join(path, " ")]...)
	for i := range command.Subcommands {
		attachAliases(&command.Subcommands[i], path)
	}
}
