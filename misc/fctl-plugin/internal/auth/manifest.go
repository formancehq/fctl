package auth

import (
	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

func manifest() pluginsdk.Manifest {
	return annotateAliases(pluginsdk.Manifest{
		Name: "auth", Version: metadata.Version, Service: "auth", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{
			Use: "auth", Short: "Manage the auth server", Service: "auth", Target: "stack",
			Subcommands: []pluginsdk.CommandSpec{clientsCommand(), usersCommand()},
		},
	})
}

func clientsCommand() pluginsdk.CommandSpec {
	create := leaf("create [name]", "Create an OAuth2 client", 0, 1)
	create.Flags = clientFlags()
	create.Inputs = []pluginsdk.InputSpec{nameInput("Client name", 0), flagInput("Description", "description"), boolInput("Public client", "public"), boolInput("Trusted client", "trusted"), flagInput("Redirect URIs (CSV)", "redirect-uri"), flagInput("Post logout redirect URIs (CSV)", "post-logout-redirect-uri"), flagInput("Client scopes (CSV)", "client-scopes")}
	confirm(&create)
	show := leaf("show <client-id>", "Show an OAuth2 client", 1, 1)
	show.Inputs = []pluginsdk.InputSpec{clientInput()}
	update := leaf("update <client-id>", "Update an OAuth2 client, preserving omitted options", 1, 1)
	update.Flags = clientFlags()
	// Optional fields must not acquire defaults in an update form. --data or
	// explicit flags select changes; the selector only supplies the client ID.
	update.Inputs = []pluginsdk.InputSpec{clientInput()}
	confirm(&update)
	remove := leaf("delete <client-id>", "Delete an OAuth2 client", 1, 1)
	remove.Inputs = []pluginsdk.InputSpec{clientInput()}
	confirm(&remove)
	return pluginsdk.CommandSpec{
		Use: "clients", Short: "Manage Auth clients",
		Subcommands: []pluginsdk.CommandSpec{create, leaf("list", "List OAuth2 clients", 0, 0), show, update, remove, secretsCommand(), usersCommand()},
	}
}

func usersCommand() pluginsdk.CommandSpec {
	show := leaf("show <user-id>", "Show an Auth user", 1, 1)
	show.Inputs = []pluginsdk.InputSpec{{
		Title: "User", Kind: "select", Argument: new(0), Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"auth", "users", "list"}, ValueField: "id", LabelFields: []string{"email", "subject", "id"}, EmptyMessage: "No Auth users are available."},
	}}
	return pluginsdk.CommandSpec{Use: "users", Short: "Manage Auth users", Subcommands: []pluginsdk.CommandSpec{leaf("list", "List Auth users", 0, 0), show}}
}

func secretsCommand() pluginsdk.CommandSpec {
	create := leaf("create <client-id> [secret-name]", "Create an OAuth2 client secret", 1, 2)
	create.Flags = []pluginsdk.FlagSpec{dataFlag(), {Name: "name", Type: "string", Usage: "Secret name"}}
	create.Inputs = []pluginsdk.InputSpec{clientInput(), nameInput("Secret name", 1)}
	confirm(&create)
	remove := leaf("delete <client-id> <secret-id>", "Delete an OAuth2 client secret", 2, 2)
	remove.Inputs = []pluginsdk.InputSpec{clientInput(), {Title: "Secret ID", Kind: "input", Argument: new(1), Required: true, Description: "Inspect auth clients show CLIENT_ID for secret IDs."}}
	confirm(&remove)
	return pluginsdk.CommandSpec{Use: "secrets", Short: "Manage client secrets", Subcommands: []pluginsdk.CommandSpec{create, remove}}
}

func leaf(use, short string, minArgs, maxArgs int) pluginsdk.CommandSpec {
	return pluginsdk.CommandSpec{Use: use, Short: short, Runnable: true, Args: pluginsdk.ArgsSpec{Min: minArgs, Max: maxArgs}}
}

func dataFlag() pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: "data", Type: "string", Body: true, Usage: "JSON object, @file or - for stdin"}
}

func clientFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{
		dataFlag(),
		{Name: "name", Type: "string", Usage: "Client name"},
		{Name: "public", Type: "bool", Default: "false", Usage: "Mark the client as public"},
		{Name: "trusted", Type: "bool", Default: "false", Usage: "Mark the client as trusted"},
		{Name: "description", Type: "string", Usage: "Client description"},
		{Name: "redirect-uri", Type: "string", Usage: "Comma-separated redirect URIs; empty clears the list"},
		{Name: "post-logout-redirect-uri", Type: "string", Usage: "Comma-separated post logout redirect URIs; empty clears the list"},
		{Name: "client-scopes", Type: "string", Usage: "Comma-separated client scopes; empty clears the list"},
	}
}

func confirm(command *pluginsdk.CommandSpec) {
	command.Confirm = true
	command.Flags = append(command.Flags, pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false", Usage: "Confirm this mutation"})
}

func clientInput() pluginsdk.InputSpec {
	return pluginsdk.InputSpec{
		Title: "OAuth2 client", Kind: "select", Argument: new(0), Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"auth", "clients", "list"}, ValueField: "id", LabelFields: []string{"name", "id"}, EmptyMessage: "No OAuth2 clients are available; create one with auth clients create."},
	}
}

func nameInput(title string, index int) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", Flag: "name", Required: true, AlternativeArgument: new(index), AlternativeFlag: "data"}
}

func flagInput(title, flag string) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", Flag: flag, AlternativeFlag: "data"}
}

func boolInput(title, flag string) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "confirm", ValueType: "bool", Flag: flag, Default: "false", AlternativeFlag: "data"}
}
