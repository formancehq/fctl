package auth

import "github.com/formancehq/fctl/v4/pkg/pluginsdk"

func manifest() pluginsdk.Manifest {
	m := pluginsdk.Manifest{
		Name: "auth", Version: "0.1.0", Service: "auth", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "auth", Target: "stack", Short: "Manage Auth clients, secrets and users", Subcommands: []pluginsdk.CommandSpec{
			leaf("info", "Show Auth server information", 0, false),
			leaf("discovery", "Show OpenID Connect discovery configuration", 0, false),
			{Use: "clients", Short: "Manage OAuth2 clients", Subcommands: []pluginsdk.CommandSpec{
				leaf("list", "List OAuth2 clients", 0, false),
				leaf("show CLIENT", "Show a client and its secret metadata", 1, false),
				bodyCommand("create", "Create an OAuth2 client", 0, false, false),
				bodyCommand("update CLIENT", "Update client options, preserving omitted fields", 1, true, false),
				leaf("delete CLIENT", "Delete an OAuth2 client", 1, true),
				{Use: "secrets", Short: "Manage client secrets (clear values are returned only at creation)", Subcommands: []pluginsdk.CommandSpec{
					leaf("list CLIENT", "List secret metadata for a client", 1, false),
					bodyCommand("create CLIENT", "Create a secret; save the returned clear value", 1, false, true),
					leaf("delete CLIENT SECRET", "Delete a client secret", 2, true),
				}},
			}},
			{Use: "users", Short: "Read Auth users", Subcommands: []pluginsdk.CommandSpec{
				leaf("list", "List users", 0, false), leaf("show USER", "Show a user", 1, false),
			}},
		}},
	}
	addInputs(&m.Root, nil)
	return m
}
func leaf(use, short string, args int, confirm bool) pluginsdk.CommandSpec {
	spec := pluginsdk.CommandSpec{Use: use, Short: short, Args: pluginsdk.ArgsSpec{Min: args, Max: args}, Runnable: true, Confirm: confirm}
	if confirm {
		spec.Flags = []pluginsdk.FlagSpec{{Name: "confirm", Type: "bool", Default: "false", Usage: "Confirm this destructive operation"}}
	}
	return spec
}
func bodyCommand(use, short string, args int, confirm, secret bool) pluginsdk.CommandSpec {
	spec := leaf(use, short, args, confirm)
	spec.Flags = append(spec.Flags, pluginsdk.FlagSpec{Name: "data", Type: "string", Usage: "JSON object; inline, @file, or - for stdin", Required: true, Body: true})
	spec.Long = short + ".\nOptions: name, description, public, trusted, redirectUris, postLogoutRedirectUris, scopes, metadata."
	if secret {
		spec.Long = short + ".\nJSON object with required name and optional metadata."
	} else if confirm {
		spec.Long += "\nOmitted options are preserved. Explicit false, empty arrays and empty objects clear options. Requires --confirm."
	} else {
		spec.Long += "\nThe name field is required. Create a secret separately with clients secrets create."
	}
	return spec
}
