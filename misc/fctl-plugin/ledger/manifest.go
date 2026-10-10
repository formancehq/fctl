package ledger

import "github.com/formancehq/fctl/pkg/pluginsdk"

func manifest() pluginsdk.Manifest {
	m := pluginsdk.Manifest{
		Name: "ledger", Service: "ledger", Version: "1.0.0", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{
			Use: "ledger", Short: "Manage ledgers with the historical v1/v2 API", Target: "stack",
			Flags: []pluginsdk.FlagSpec{{Name: "ledger", Type: "string", Default: "default", Usage: "Specific ledger", Persistent: true}},
			Subcommands: []pluginsdk.CommandSpec{
				command("create <name>", "Create a ledger (Ledger v2 or later)", 1, 1, true,
					[]pluginsdk.FlagSpec{stringFlag("bucket", "Bucket in which to install the ledger"), pairsFlag("features", "Features (name=value; Ledger v2.2 or later)"), pairsFlag("metadata", "Ledger metadata")},
					argument("Ledger name", 0)),
				withLedger(command("send [<source>] <destination> <amount> <asset>", "Send from one account to another (source defaults to world)", 3, 4, true,
					[]pluginsdk.FlagSpec{pairsFlag("metadata", "Transaction metadata"), stringFlag("reference", "Transaction reference")},
					argument("Destination account", 0), argument("Amount (integer)", 1), argument("Asset", 2))),
				withLedger(command("stats", "Read ledger stats", 0, 0, false, nil)),
				command("server-infos", "Read server info", 0, 0, false, nil),
				command("list", "List ledgers (Ledger v2 or later)", 0, 0, false, pagination("0")),
				command("set-metadata <ledger-name> <key>=<value>...", "Set metadata on a ledger", 2, unlimitedArgs, true, nil, ledgerArgument(0), argument("Metadata key=value", 1)),
				command("delete-metadata <ledger-name> <key>", "Delete metadata from a ledger", 2, unlimitedArgs, true, nil, ledgerArgument(0), argument("Metadata key", 1)),
				withLedger(command("export", "Export ledger logs as JSON; file output belongs to the host", 0, 0, false,
					[]pluginsdk.FlagSpec{stringFlag("file", "Output file handled by the host; plugin always returns JSON")})),
				command("import <ledger-name> [<file-path>]", "Import logs supplied by the host in --data or SDK Body", 1, 2, false,
					[]pluginsdk.FlagSpec{bodyFlag("JSON array of logs or JSON string containing NDJSON; @file/- is read by the host"), stringFlag("file", "Historical source hint; the host must read it into Body"), boolFlag("resume-from-last-log", "Resume after the last imported log")},
					ledgerArgument(0), bodyInput("Logs", "JSON array of logs or JSON string containing NDJSON")),
				{Use: "accounts", Short: "Manage accounts", Subcommands: []pluginsdk.CommandSpec{
					withLedger(command("list", "List accounts", 0, 0, false, append([]pluginsdk.FlagSpec{pairsFlag("metadata", "Filter account metadata")}, pagination("0")...))),
					withLedger(command("show <address>", "Show an account", 1, 1, false, nil, accountArgument(0))),
					withLedger(command("set-metadata <address> <key>=<value>...", "Set account metadata", 2, unlimitedArgs, true, nil, accountArgument(0), argument("Metadata key=value", 1))),
					withLedger(command("delete-metadata <address> <key>", "Delete account metadata", 2, unlimitedArgs, true, nil, accountArgument(0), argument("Metadata key", 1))),
				}},
				{Use: "transactions", Short: "Manage transactions", Subcommands: []pluginsdk.CommandSpec{
					withLedger(command("list", "List transactions", 0, 0, false, append([]pluginsdk.FlagSpec{
						stringFlag("account", "Filter by account"), stringFlag("dst", "Filter by destination"), stringFlag("src", "Filter by source"),
						stringFlag("start", "Inclusive start time (RFC3339)"), stringFlag("end", "Exclusive end time (RFC3339)"),
						stringFlag("reference", "Filter by reference"), pairsFlag("metadata", "Filter transaction metadata"),
					}, pagination("5")...))),
					withLedger(command("num [<filename>|-]", "Execute Numscript supplied by the host in --data or SDK Body", 0, 1, true,
						[]pluginsdk.FlagSpec{bodyFlag("JSON string containing Numscript, {\"plain\":...}, or {\"script\":{\"plain\":...,\"vars\":...}}; source files are read by the host"),
							pairsFlag("amount-var", "Amount variables name=amount/asset"), pairsFlag("portion-var", "Portion variables name=value"), pairsFlag("account-var", "Account variables name=value"),
							pairsFlag("metadata", "Transaction metadata"), stringFlag("reference", "Transaction reference"), stringFlag("timestamp", "Transaction timestamp (RFC3339)")},
						pluginsdk.InputSpec{Title: "Numscript", Kind: "text", BodyPointer: "/script/plain", ValueType: "string", Required: true})),
					withLedger(command("revert <transaction-id>", "Revert a transaction; accepts last or lastN", 1, 1, true,
						[]pluginsdk.FlagSpec{boolFlag("at-effective-date", "Use original timestamp via Ledger v2"), boolFlag("force", "Disable balance checks")}, transactionArgument(0))),
					withLedger(command("show <transaction-id>", "Show a transaction; accepts last or lastN", 1, 1, false, nil, transactionArgument(0))),
					withLedger(command("set-metadata <transaction-id> <key>=<value>...", "Set transaction metadata; accepts last or lastN", 2, unlimitedArgs, true, nil, transactionArgument(0), argument("Metadata key=value", 1))),
					withLedger(command("delete-metadata <transaction-id> <key>", "Delete transaction metadata; accepts last or lastN", 2, unlimitedArgs, true, nil, transactionArgument(0), argument("Metadata key", 1))),
				}},
				{Use: "schemas", Short: "Manage ledger schemas", Subcommands: []pluginsdk.CommandSpec{
					withLedger(command("insert <version> [<source>]", "Insert a schema supplied as JSON Body; host reads JSON/YAML files or URLs", 1, 2, true,
						[]pluginsdk.FlagSpec{bodyFlag("Schema JSON object, @file or -; host converts YAML to JSON")}, argument("Schema version", 0),
						pluginsdk.InputSpec{Title: "Chart of accounts", Kind: "text", BodyPointer: "/chart", ValueType: "json", Required: true})),
					withLedger(command("get <version>", "Get a schema; output presentation belongs to the host", 1, 1, false,
						[]pluginsdk.FlagSpec{{Name: "format", Type: "string", Default: "json", Usage: "Host presentation hint: json, yaml or yml; SDK output remains JSON"}}, schemaArgument(0))),
					withLedger(command("list", "List ledger schemas", 0, 0, false, pagination("15"))),
				}},
				{Use: "volumes", Short: "Get volumes and balances", Subcommands: []pluginsdk.CommandSpec{
					withLedger(command("list", "List volumes and balances for a time period", 0, 0, false, append([]pluginsdk.FlagSpec{
						stringFlag("end-time", "PIT (RFC3339)"), stringFlag("start-time", "OOT (RFC3339)"), boolFlag("insertion-date", "Use insertion date"),
						{Name: "group-by", Type: "uint32", Default: "0", Usage: "Group by address segment level"}, stringFlag("address", "Filter account address"), pairsFlag("metadata", "Filter account metadata"),
					}, pagination("10")...))),
				}},
			},
		},
	}
	return annotateFiles(m)
}

const unlimitedArgs = 1<<31 - 1

func command(use, short string, minArgs, maxArgs int, confirm bool, flags []pluginsdk.FlagSpec, inputs ...pluginsdk.InputSpec) pluginsdk.CommandSpec {
	if confirm {
		flags = append(flags, boolFlag("confirm", "Confirm this operation"))
	}
	return pluginsdk.CommandSpec{Use: use, Short: short, Args: pluginsdk.ArgsSpec{Min: minArgs, Max: maxArgs}, Runnable: true, Confirm: confirm, Flags: flags, Inputs: inputs}
}

func stringFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "string", Usage: usage}
}

func pairsFlag(name, usage string) pluginsdk.FlagSpec {
	return stringFlag(name, usage+`; use comma-separated key=value pairs or a JSON string array, e.g. ["key=value,with,commas"]`)
}

func boolFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "bool", Default: "false", Usage: usage}
}

func bodyFlag(usage string) pluginsdk.FlagSpec {
	flag := stringFlag("data", usage)
	flag.Body = true
	return flag
}

func pagination(defaultSize string) []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{stringFlag("cursor", "Opaque pagination cursor"), {Name: "page-size", Type: "uint32", Default: defaultSize, Usage: "Page size; 0 uses the server default when omitted"}}
}

func argument(title string, index int) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", Argument: new(index), ValueType: "string", Required: true}
}

func bodyInput(title, description string) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Description: description, Kind: "text", Flag: "data", ValueType: "json", Required: true}
}

func ledgerArgument(index int) pluginsdk.InputSpec {
	i := argument("Ledger", index)
	i.Kind = "select"
	i.Source = &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "list"}, ValueField: "name", LabelFields: []string{"name"}, EmptyMessage: "No ledgers available"}
	return i
}

func withLedger(c pluginsdk.CommandSpec) pluginsdk.CommandSpec {
	i := ledgerArgument(0)
	i.Argument, i.Flag, i.Default = nil, "ledger", "default"
	c.Inputs = append([]pluginsdk.InputSpec{i}, c.Inputs...)
	return c
}

func accountArgument(index int) pluginsdk.InputSpec {
	i := argument("Account", index)
	i.Kind = "select"
	i.Source = &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "accounts", "list"}, Flags: map[string]string{"ledger": "$ledger"}, ValueField: "address", LabelFields: []string{"address"}, EmptyMessage: "No accounts available"}
	return i
}

func transactionArgument(index int) pluginsdk.InputSpec {
	i := argument("Transaction ID (or last/lastN)", index)
	i.Kind = "select"
	i.Source = &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "transactions", "list"}, Flags: map[string]string{"ledger": "$ledger"}, ValueField: "txid", LabelFields: []string{"txid", "reference"}, EmptyMessage: "No transactions available"}
	return i
}

func schemaArgument(index int) pluginsdk.InputSpec {
	i := argument("Schema version", index)
	i.Kind = "select"
	i.Source = &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "schemas", "list"}, Flags: map[string]string{"ledger": "$ledger"}, ValueField: "version", LabelFields: []string{"version"}, EmptyMessage: "No schemas available"}
	return i
}
