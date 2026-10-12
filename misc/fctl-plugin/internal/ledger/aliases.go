package ledger

import (
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

var historicalAliases = map[string][]string{
	"ledger": {"l"}, "ledger create": {"c", "cr"}, "ledger list": {"l", "ls"}, "ledger send": {"s", "se"}, "ledger stats": {"st"}, "ledger server-infos": {"si"},
	"ledger set-metadata": {"sm", "set-meta"}, "ledger delete-metadata": {"dm", "del-meta"},
	"ledger accounts": {"acc", "a", "ac", "account"}, "ledger accounts list": {"ls", "l"}, "ledger accounts show": {"sh", "s"}, "ledger accounts set-metadata": {"sm", "set-meta"}, "ledger accounts delete-metadata": {"dm", "del-meta"},
	"ledger transactions": {"t", "txs", "tx"}, "ledger transactions list": {"ls", "l"}, "ledger transactions show": {"sh"}, "ledger transactions set-metadata": {"sm", "set-meta"}, "ledger transactions delete-metadata": {"dm", "del-meta"},
	"ledger schemas": {"schema", "sc"}, "ledger schemas get": {"g", "show"}, "ledger schemas insert": {"i", "create"}, "ledger schemas list": {"ls", "l"},
	"ledger volumes": {"vol", "volume", "vols", "vlm"}, "ledger volumes list": {"ls", "l"},
}

func annotateAliases(command *pluginsdk.CommandSpec, path []string) {
	path = append(append([]string{}, path...), pluginsdk.CommandName(*command))
	command.Aliases = append([]string{}, historicalAliases[strings.Join(path, " ")]...)
	for i := range command.Subcommands {
		annotateAliases(&command.Subcommands[i], path)
	}
}
