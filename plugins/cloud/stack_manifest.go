package cloud

import "github.com/formancehq/fctl/pkg/pluginsdk"

func stackManifest() pluginsdk.CommandSpec {
	create := stackCommand("create [NAME]", "Create a stack", 0, 1, false)
	create.Flags = append(stackBodyFlags(), stackStringFlag("name", "", "Name of the new stack"), stackStringFlag("region", "", "Region ID; required when more than one region is available"), stackStringFlag("version", "v4.0", "Exact catalog version to create (defaults to v4.0; no fallback)"))
	create.Flags = append(create.Flags, stackWaitFlags()...)
	show := stackCommand("show [STACK]", "Show stack metadata from Membership", 0, 1, false)
	show.Flags = []pluginsdk.FlagSpec{stackStringFlag("name", "", "Find a stack by its exact name instead of its ID")}
	update := stackCommand("update [STACK]", "Update stack name or metadata; omitted fields are preserved", 0, 1, false)
	update.Flags = append(stackBodyFlags(), stackStringFlag("name", "", "New stack name"))
	remove := stackCommand("delete [STACK]", "Delete a stack after explicit confirmation", 0, 1, true)
	remove.Flags = append(remove.Flags, stackBoolFlag("force", "Delete even when the stack is not empty"))
	disable := stackCommand("disable [STACK]", "Disable a stack after explicit confirmation", 0, 1, true)
	restore := stackCommand("restore [STACK]", "Restore a deleted stack after explicit confirmation", 0, 1, true)
	restore.Flags = append(restore.Flags, stackWaitFlags()...)
	upgrade := stackCommand("upgrade [STACK] [VERSION]", "Upgrade a stack to a catalog version", 0, 2, true)
	upgrade.Flags = append(upgrade.Flags, stackBodyFlags()...)
	upgrade.Flags = append(upgrade.Flags, stackStringFlag("version", "v4.0", "Exact target catalog version (defaults to v4.0; no fallback)"))
	upgrade.Flags = append(upgrade.Flags, stackWaitFlags()...)
	list := stackCommand("list", "List stacks in the selected organization", 0, 0, false)
	list.Flags = []pluginsdk.FlagSpec{stackBoolFlag("all", "Include disabled and deleted stacks"), stackBoolFlag("deleted", "Include deleted stacks (deprecated; prefer --all)")}
	history := stackCommand("history [STACK]", "Read organization audit logs for this stack", 0, 1, false)
	history.Flags = []pluginsdk.FlagSpec{
		{Name: "page-size", Type: "uint32", Default: "10", Usage: "Number of logs per page (1 through 1000)"},
		stackStringFlag("cursor", "", "Opaque pagination cursor; cannot be combined with other log filters"),
		stackStringFlag("action", "", "Stack action filter, such as stacks.create"),
		stackStringFlag("user-id", "", "Actor ID; SYSTEM selects system logs"),
		stackStringFlag("data", "", "Log data filter as key=value (key is a JSONB text path)"),
	}
	moduleEnable := stackCommand("enable MODULE [STACK]", "Enable a module on the selected stack", 1, 2, false)
	moduleDisable := stackCommand("disable MODULE [STACK]", "Disable a module after explicit confirmation", 1, 2, true)
	link := stackCommand("link USER [STACK]", "Link a user to a stack policy", 1, 2, false)
	link.Flags = append(stackBodyFlags(), stackStringFlag("policy-id", "", "Positive stack policy ID (integer)"))
	return pluginsdk.CommandSpec{Use: "stack", Short: "Manage stacks through Membership", Subcommands: []pluginsdk.CommandSpec{
		create, list, show, update, remove, disable,
		stackCommand("enable [STACK]", "Enable a disabled stack after explicit confirmation", 0, 1, true),
		restore, upgrade, history,
		stackCommand("info [STACK]", "Show Membership stack metadata; does not contact the gateway", 0, 1, false),
		stackCommand("version [STACK]", "Show the desired version recorded by Membership; component versions require gateway access", 0, 1, false),
		{Use: "modules", Short: "Manage stack modules", Subcommands: []pluginsdk.CommandSpec{stackCommand("list [STACK]", "List stack modules", 0, 1, false), moduleEnable, moduleDisable}},
		{Use: "users", Short: "Manage stack user access", Subcommands: []pluginsdk.CommandSpec{stackCommand("list [STACK]", "List stack user access", 0, 1, false), link, stackCommand("unlink USER [STACK]", "Remove stack user access after explicit confirmation", 1, 2, true)}},
	}}
}

func stackCommand(use, short string, minArgs, maxArgs int, confirm bool) pluginsdk.CommandSpec {
	spec := pluginsdk.CommandSpec{Use: use, Short: short, Runnable: true, Args: pluginsdk.ArgsSpec{Min: minArgs, Max: maxArgs}, Confirm: confirm}
	if confirm {
		spec.Flags = []pluginsdk.FlagSpec{stackBoolFlag("confirm", "Confirm this operation")}
	}
	return spec
}

func stackStringFlag(name, value, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "string", Default: value, Usage: usage}
}

func stackBoolFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "bool", Default: "false", Usage: usage}
}

func stackBodyFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{{Name: "data", Type: "string", Body: true, Usage: "JSON request object: inline JSON, @file or - for stdin; read by the host"}}
}

func stackWaitFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{stackBoolFlag("no-wait", "Return after Membership accepts the operation"), stackStringFlag("wait-timeout", "10m", "Maximum readiness wait, such as 30s or 10m; cancellation stops polling")}
}
