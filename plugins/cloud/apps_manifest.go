package cloud

import "github.com/formancehq/fctl/v4/pkg/pluginsdk"

// appsManifest targets the Deploy audience selected by the host, not Membership.
func appsManifest() pluginsdk.CommandSpec {
	str := func(name string) pluginsdk.FlagSpec {
		return pluginsdk.FlagSpec{Name: name, Type: "string", Usage: name}
	}
	id := func(name string) pluginsdk.FlagSpec { f := str(name); f.Required = true; return f }
	body := pluginsdk.FlagSpec{Name: "data", Type: "string", Body: true, Usage: "JSON body: inline, @file or - for stdin; YAML upload: {\"yaml\":\"...\"}"}
	page := []pluginsdk.FlagSpec{str("cursor"), {Name: "page-size", Type: "uint32", Default: "100", Usage: "Positive page size"}}
	wait := func(defaultValue string) []pluginsdk.FlagSpec {
		return []pluginsdk.FlagSpec{{Name: "wait", Type: "bool", Default: defaultValue, Usage: "Wait for a terminal deployment status"}, {Name: "wait-timeout", Type: "string", Default: "30m", Usage: "Positive bounded wait duration (maximum 24h)"}}
	}
	leaf := func(name string, confirm bool, flags ...pluginsdk.FlagSpec) pluginsdk.CommandSpec {
		s := pluginsdk.CommandSpec{Use: name, Short: "Deploy service: " + name, Runnable: true, Confirm: confirm, Flags: flags}
		if confirm {
			s.Flags = append(s.Flags, pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false", Usage: "Confirm destructive operation"})
		}
		return s
	}
	group := func(name string, children ...pluginsdk.CommandSpec) pluginsdk.CommandSpec {
		return pluginsdk.CommandSpec{Use: name, Short: "Manage app " + name, Subcommands: children}
	}
	return pluginsdk.CommandSpec{Use: "apps", Short: "Manage experimental Deploy apps", Service: "cloud-apps", Flags: []pluginsdk.FlagSpec{
		{Name: "experimental", Type: "bool", Default: "false", Persistent: true, RequireTrue: true, Usage: "Enable experimental Deploy commands"},
		{Name: "deploy-app-alias", Type: "string", Default: "deploy", Persistent: true, Usage: "Membership application alias used by the host to resolve Deploy access"},
	}, Subcommands: []pluginsdk.CommandSpec{
		leaf("list", false, page...), leaf("show", false, id("id")), leaf("create", false, body, str("name"), str("stack-id")),
		leaf("delete", true, append([]pluginsdk.FlagSpec{id("id")}, wait("false")...)...),
		leaf("bind-manifest", false, id("app-id"), body, str("manifest-id")), leaf("unbind-manifest", true, id("app-id")),
		group("manifests", leaf("list", false, page...), leaf("show", false, id("id")), leaf("create", false, body, str("name"), str("path")), leaf("update", false, id("id"), body, str("name")), leaf("delete", true, id("id")), leaf("download", false, id("id"), pluginsdk.FlagSpec{Name: "version", Type: "string", Default: "latest", Usage: "Positive version or latest"}, str("out")),
			group("versions", leaf("list", false, append([]pluginsdk.FlagSpec{id("manifest-id")}, page...)...), leaf("push", false, id("manifest-id"), body, str("path")), leaf("show", false, id("manifest-id"), pluginsdk.FlagSpec{Name: "version", Type: "string", Default: "latest", Usage: "Positive version or latest"}))),
		group("deployments", leaf("list", false, append(page, str("app-id"))...), leaf("show", false, id("id"), pluginsdk.FlagSpec{Name: "include-state", Type: "bool", Default: "false", Usage: "Include live state"}), leaf("logs", false, id("id")), leaf("download", false, id("id"), str("out")), leaf("create", false, append([]pluginsdk.FlagSpec{body, str("app-id"), str("manifest-id"), str("manifest-version")}, wait("true")...)...)),
		group("variables", leaf("list", false, append([]pluginsdk.FlagSpec{id("id")}, page...)...), leaf("create", false, id("id"), body, str("key"), str("value"), str("description")), leaf("delete", true, id("app-id"), id("id"))),
	}}
}
