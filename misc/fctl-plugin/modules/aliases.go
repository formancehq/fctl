package modules

import (
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// Only unambiguous historical aliases are retained. Payments historically used
// "p" for both payments and pools; it belongs to payments in this bundle.
// Conflicting "cr" belongs to wallet create, "r" belongs to transfer retry,
// and "h" remains reserved by the host for help.
var historicalGroups = map[string]map[string][]string{
	"orchestration": {"": {"orch", "or"}, "instances": {"ins", "i"}, "workflows": {"w", "work"}, "triggers": {"trig", "t"}, "triggers occurrences": {"oc", "o"}, "workflows run": {"r"}},
	"payments": {"accounts": {"acc", "a", "ac", "account"}, "bank-accounts": {"bank_accounts", "bacc", "ba", "bac", "baccount"}, "payments": {"p"}, "pools": {"pool"},
		"transfer-initiation": {"transfer_initiation", "payment_initiations", "payment-initiations", "ti"}, "transfer-initiation update-status": {"update_status", "u"}, "connectors": {"c", "co", "con"},
		"connectors get-config": {"getconfig", "getconf", "gc", "get", "g"}, "connectors update-config": {"uc"}, "connectors list-available": {"la"}, "connectors schedules": {"sch"},
		"connectors schedules instances": {"inst"}, "connectors install": {"i"}, "connectors uninstall": {"u", "un"},
		"bank-accounts forward": {"fo", "f"}, "bank-accounts update-metadata": {"um", "update-meta"}, "payments set-metadata": {"sm", "set-meta"},
		"pools add-account": {"add", "a"}, "pools remove-account": {"remove", "rm"}, "pools update-query": {"uq"},
		"transfer-initiation approve": {"a"}, "transfer-initiation reject": {"rj"}, "transfer-initiation retry": {"r"}, "transfer-initiation reverse": {"re"}, "orders": {"o"}, "conversions": {"cv"}, "tasks": {"t"}},
	"reconciliation": {"policies": {"p"}, "rules": {"rule"}, "rules evaluate": {"run"}, "rules update": {"patch"}, "rules get": {"show", "sh", "s"}, "evaluations get": {"show", "sh", "s"}, "alerts get": {"show", "sh", "s"}, "policies reconcile": {"r"}, "evaluations": {"evaluation", "evals"}, "alerts": {"alert"}},
	"wallets":        {"": {"wal", "wa", "wallet"}, "balances": {"balance", "bls", "bal"}, "holds": {"hold", "h"}, "holds confirm": {"c", "conf"}, "holds void": {"v"}, "debit": {"deb"}, "transactions": {"transaction", "tx", "txs"}},
	"webhooks":       {"": {"web", "wh"}, "deliveries": {"delivery", "dlvs"}, "change-secret": {"cs"}, "activate": {"ac", "a"}, "deactivate": {"deac"}, "deliveries show": {"get"}},
}

var commonAliases = map[string][]string{"list": {"ls", "l"}, "get": {"sh", "s"}, "show": {"sh", "s"}, "create": {"cr", "c"}, "delete": {"del", "d"}, "update": {"up"}}

func applyAliases(name string, command *pluginsdk.CommandSpec, path []string) {
	if aliases, exists := historicalGroups[name][strings.Join(path, " ")]; exists {
		command.Aliases = aliases
	} else if command.Runnable {
		command.Aliases = commonAliases[pluginsdk.CommandName(*command)]
	}
	for i := range command.Subcommands {
		child := &command.Subcommands[i]
		applyAliases(name, child, append(append([]string{}, path...), pluginsdk.CommandName(*child)))
	}
}
