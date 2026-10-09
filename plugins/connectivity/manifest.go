package connectivity

import "github.com/formancehq/fctl/pkg/pluginsdk"

func manifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "connectivity", Version: "0.1.0", Service: "connectivity", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "connectivity", Target: "stack", Short: "Manage connectors and ingestion instances", Subcommands: []pluginsdk.CommandSpec{
			leaf("info", "Show Connectivity server information", 0, nil),
			leaf("health", "Check Connectivity health", 0, nil),
			{Use: "query", Short: "Inspect supported query fields and operators", Subcommands: []pluginsdk.CommandSpec{leaf("capabilities", "Show the query contract", 0, nil)}},
			{Use: "connectors", Short: "Browse the connector catalogue", Subcommands: []pluginsdk.CommandSpec{
				list("list", "List published connectors", 0, nil),
				facets(),
				leaf("show CONNECTOR", "Show a connector", 1, []pluginsdk.InputSpec{connectorInput()}),
				{Use: "versions", Short: "Browse connector versions and configuration schemas", Subcommands: []pluginsdk.CommandSpec{
					list("list CONNECTOR", "List connector versions", 1, []pluginsdk.InputSpec{connectorInput()}),
					leaf("show CONNECTOR VERSION", "Show a version or channel alias and its configuration schema", 2, []pluginsdk.InputSpec{connectorInput(), versionInput()}),
				}},
			}},
			{Use: "instances", Short: "Manage connector instances in this Connectivity namespace", Subcommands: []pluginsdk.CommandSpec{
				list("list", "List connector instances", 0, nil),
				leaf("show INSTANCE", "Show instance configuration and reconciliation status", 1, []pluginsdk.InputSpec{instanceInput()}),
				body("create", "Create a connector instance", 0, false, createInputs()),
				body("replace INSTANCE", "Replace the full desired spec, preserving metadata and status", 1, true, []pluginsdk.InputSpec{instanceInput(), {Title: "Complete instance spec", Description: "JSON object replacing the entire spec; include connector, ledger and configuration", Kind: "text", BodyPointer: "/spec", ValueType: "json", Required: true}}),
				body("patch INSTANCE", "Apply a JSON merge patch directly to spec; null removes a field", 1, false, patchInputs()),
				deletion(),
			}},
		}},
	}
}

func leaf(use, short string, args int, inputs []pluginsdk.InputSpec) pluginsdk.CommandSpec {
	return pluginsdk.CommandSpec{Use: use, Short: short, Runnable: true, Args: pluginsdk.ArgsSpec{Min: args, Max: args}, Inputs: inputs}
}

func list(use, short string, args int, inputs []pluginsdk.InputSpec) pluginsdk.CommandSpec {
	c := leaf(use, short, args, inputs)
	c.Flags = []pluginsdk.FlagSpec{
		{Name: "page-size", Type: "uint32", Default: "15", Usage: "Records per page (1-100)"},
		{Name: "cursor", Type: "string", Usage: "Opaque next cursor from the previous page"},
		queryFlag(),
	}
	return c
}

func queryFlag() pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: "query", Type: "string", Usage: "JSON query expression; see connectivity query capabilities"}
}

func facets() pluginsdk.CommandSpec {
	c := leaf("facets", "Count catalogue facets, optionally filtered by query", 0, nil)
	c.Flags = []pluginsdk.FlagSpec{queryFlag()}
	return c
}

func body(use, short string, args int, confirm bool, inputs []pluginsdk.InputSpec) pluginsdk.CommandSpec {
	c := leaf(use, short, args, inputs)
	c.Flags = []pluginsdk.FlagSpec{{Name: "data", Type: "string", Required: true, Body: true, Usage: "JSON object (max 1 MiB); inline, @file, or - for stdin"}}
	c.Confirm = confirm
	if confirm {
		c.Flags = append(c.Flags, confirmation())
	}
	return c
}

func confirmation() pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false", Usage: "Confirm this destructive operation"}
}

func deletion() pluginsdk.CommandSpec {
	c := leaf("delete INSTANCE", "Delete a connector instance", 1, []pluginsdk.InputSpec{instanceInput()})
	c.Confirm, c.Flags = true, []pluginsdk.FlagSpec{confirmation()}
	return c
}
