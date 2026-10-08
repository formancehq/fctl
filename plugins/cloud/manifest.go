package cloud

import "github.com/formancehq/fctl/v4/pkg/pluginsdk"

func manifest() pluginsdk.Manifest {
	m := pluginsdk.Manifest{Name: "cloud", Version: "0.1.0", Service: "cloud", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{
		Use: "cloud", Short: "Manage Formance Cloud", Subcommands: []pluginsdk.CommandSpec{meManifest(), organizationsManifest(), regionsManifest(), stackManifest(), appsManifest()},
	}}
	addCloudInteraction(&m.Root, nil, cloudInteractionInputs())
	return m
}
func leaf(use, short string, args int, confirm bool) pluginsdk.CommandSpec {
	s := pluginsdk.CommandSpec{Use: use, Short: short, Args: pluginsdk.ArgsSpec{Min: args, Max: args}, Runnable: true, Confirm: confirm}
	if confirm {
		s.Flags = append(s.Flags, pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false", Usage: "Confirm this destructive operation"})
	}
	return s
}
func leafBody(use, short string, args int, confirm bool) pluginsdk.CommandSpec {
	s := leaf(use, short, args, confirm)
	s.Flags = append(s.Flags, pluginsdk.FlagSpec{Name: "data", Type: "string", Body: true, Usage: "JSON object: inline, @file, or - for stdin"})
	return s
}
func stringFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "string", Usage: usage}
}
func boolFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "bool", Default: "false", Usage: usage}
}
func withFlags(s pluginsdk.CommandSpec, flags ...pluginsdk.FlagSpec) pluginsdk.CommandSpec {
	s.Flags = append(s.Flags, flags...)
	return s
}
func optionalArg(s pluginsdk.CommandSpec) pluginsdk.CommandSpec { s.Args.Min = 0; return s }
func paginationFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{stringFlag("cursor", "Pagination cursor"), {Name: "page-size", Type: "uint32", Default: "100", Usage: "Number of results (positive)"}}
}
