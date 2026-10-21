package command

import (
	"slices"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

var commonAliases = map[string][]string{"list": {"ls", "l"}, "get": {"sh", "s"}, "show": {"sh", "s"}, "create": {"cr", "c"}, "delete": {"del", "d"}, "update": {"up"}}

func applyAliases(command *pluginsdk.CommandSpec, path []string, aliases map[string][]string) {
	if values, exists := aliases[strings.Join(path, " ")]; exists {
		command.Aliases = slices.Clone(values)
	} else if command.Runnable {
		command.Aliases = slices.Clone(commonAliases[pluginsdk.CommandName(*command)])
	}
	for i := range command.Subcommands {
		child := &command.Subcommands[i]
		applyAliases(child, append(slices.Clone(path), pluginsdk.CommandName(*child)), aliases)
	}
}
