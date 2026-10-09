package cloud

import (
	"maps"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// Only targets are inherited. Inputs belong to runnable leaves so the host can
// fill omissions without changing the plugin's noninteractive request contract.
func addCloudInteraction(command *pluginsdk.CommandSpec, parent []string, inputs map[string][]pluginsdk.InputSpec) {
	path := append(append([]string{}, parent...), pluginsdk.CommandName(*command))
	key := strings.Join(path, " ")
	switch key {
	case "cloud", "cloud organizations list", "cloud organizations create":
		command.Target = "identity"
	case "cloud organizations", "cloud regions", "cloud stack", "cloud apps":
		command.Target = "organization"
	}
	if command.Runnable {
		command.Inputs = inputs[key]
	}
	for i := range command.Subcommands {
		addCloudInteraction(&command.Subcommands[i], path, inputs)
	}
}

func cloudInteractionInputs() map[string][]pluginsdk.InputSpec {
	inputs := organizationInteractionInputs()
	maps.Copy(inputs, regionInteractionInputs())
	maps.Copy(inputs, stackInteractionInputs())
	maps.Copy(inputs, appInteractionInputs())
	for _, action := range []string{"accept", "decline"} {
		inputs["cloud me invitations "+action] = []pluginsdk.InputSpec{
			argumentChoice("Invitation", 0, cloudChoices("cloud me invitations list", "id", "No invitations are available; inspect fctl cloud me invitations list.", "organizationName", "email", "id")),
		}
	}
	return inputs
}

func flagInput(title, flag string, required bool) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", Flag: flag, Required: required}
}

func payloadFlagInput(title, flag string) pluginsdk.InputSpec {
	input := flagInput(title, flag, true)
	input.AlternativeFlag = "data"
	return input
}

func nameInput(title string) pluginsdk.InputSpec {
	input := payloadFlagInput(title, "name")
	input.AlternativeArgument = new(0)
	return input
}

func bodyInput(title, pointer string, required bool) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", BodyPointer: pointer, Required: required}
}

func argumentInput(title string, index int) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "input", Argument: new(index), Required: true}
}

func cloudChoices(path, value, empty string, labels ...string) *pluginsdk.ChoiceSource {
	source := &pluginsdk.ChoiceSource{CommandPath: strings.Fields(path), ValueField: value, LabelFields: labels, EmptyMessage: empty}
	if strings.HasPrefix(path, "cloud apps ") {
		// Discovery must retain the user's explicit experimental opt-in.
		source.Flags = map[string]string{"experimental": "$experimental"}
	}
	return source
}

func flagChoice(title, flag string, source *pluginsdk.ChoiceSource) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "select", Flag: flag, Required: true, Source: source}
}

func payloadFlagChoice(title, flag string, source *pluginsdk.ChoiceSource) pluginsdk.InputSpec {
	input := flagChoice(title, flag, source)
	input.AlternativeFlag = "data"
	return input
}

func argumentChoice(title string, index int, source *pluginsdk.ChoiceSource) pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: title, Kind: "select", Argument: new(index), Required: true, Source: source}
}

func organizationPolicyChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud organizations policies list", "id", "No organization policies are available; create one with fctl cloud organizations policies create.", "name", "id")
}

func organizationUserChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud organizations users list", "id", "No organization users are available; invite a user with fctl cloud organizations invitations send.", "name", "email", "id")
}

func organizationInteractionInputs() map[string][]pluginsdk.InputSpec {
	inputs := map[string][]pluginsdk.InputSpec{
		"cloud organizations create":                            {nameInput("Organization name")},
		"cloud organizations policies create":                   {payloadFlagInput("Policy name", "name")},
		"cloud organizations oauth-clients create":              {payloadFlagInput("OAuth client name", "name")},
		"cloud organizations invitations send":                  {{Title: "Invitee email", Kind: "input", Flag: "email", Required: true, AlternativeArgument: new(0)}},
		"cloud organizations users link":                        {argumentInput("User ID to link", 0), payloadFlagChoice("Organization policy", "policy-id", organizationPolicyChoices())},
		"cloud organizations authentication-provider configure": providerInteractionInputs(),
		// The list contains enabled applications, not a catalog of applications
		// that could be enabled. Do not offer that list as an enable catalog.
		"cloud organizations applications enable": {argumentInput("Application ID to enable", 0)},
	}
	inputs["cloud organizations users link"][1].ValueType = "number"
	for _, action := range []string{"describe", "update", "delete", "history"} {
		inputs["cloud organizations "+action] = []pluginsdk.InputSpec{{
			Title: "Organization", Kind: "select", Context: "organization", Required: true, AlternativeArgument: new(0),
			Source: cloudChoices("cloud organizations list", "id", "No organizations are available; create one with fctl cloud organizations create.", "name", "id"),
		}}
	}
	for _, action := range []string{"show", "unlink"} {
		inputs["cloud organizations users "+action] = []pluginsdk.InputSpec{argumentChoice("Organization user", 0, organizationUserChoices())}
	}
	inputs["cloud organizations invitations delete"] = []pluginsdk.InputSpec{
		argumentChoice("Invitation", 0, cloudChoices("cloud organizations invitations list", "id", "No invitations are available; send one with fctl cloud organizations invitations send.", "email", "id")),
	}
	for _, action := range []string{"show", "update", "delete", "add-scope", "remove-scope"} {
		input := argumentChoice("Organization policy", 0, organizationPolicyChoices())
		input.ValueType = "number"
		inputs["cloud organizations policies "+action] = []pluginsdk.InputSpec{input}
	}
	for _, action := range []string{"add-scope", "remove-scope"} {
		input := argumentInput("Scope ID", 1)
		input.ValueType = "number"
		input.Description = "Positive scope ID. Inspect fctl cloud organizations policies show POLICY; the plugin has no scope catalog command."
		inputs["cloud organizations policies "+action] = append(inputs["cloud organizations policies "+action], input)
	}
	for _, action := range []string{"show", "update", "delete"} {
		inputs["cloud organizations oauth-clients "+action] = []pluginsdk.InputSpec{
			argumentChoice("OAuth client", 0, cloudChoices("cloud organizations oauth-clients list", "id", "No OAuth clients are available; create one with fctl cloud organizations oauth-clients create.", "name", "id")),
		}
	}
	for _, action := range []string{"show", "disable"} {
		inputs["cloud organizations applications "+action] = []pluginsdk.InputSpec{
			argumentChoice("Organization application", 0, cloudChoices("cloud organizations applications list", "id", "No applications are enabled; use fctl cloud organizations applications enable APPLICATION.", "name", "id")),
		}
	}
	for _, path := range []string{"cloud organizations update", "cloud organizations policies update", "cloud organizations oauth-clients update"} {
		name := payloadFlagInput("New name", "name")
		name.Description = "Enter a new name, or use --data to update other fields without renaming."
		inputs[path] = append(inputs[path], name)
	}
	return inputs
}

func providerInteractionInputs() []pluginsdk.InputSpec {
	kind := payloadFlagInput("Provider type", "type")
	kind.Kind, kind.AlternativeArgument = "select", new(0)
	for _, value := range []string{"github", "google", "microsoft", "oidc"} {
		kind.Options = append(kind.Options, pluginsdk.InputOption{Label: value, Value: value})
	}
	name := payloadFlagInput("Provider name", "name")
	name.AlternativeArgument = new(1)
	id := payloadFlagInput("OAuth client ID", "provider-client-id")
	id.AlternativeArgument = new(2)
	secret := payloadFlagInput("OAuth client secret", "provider-client-secret")
	secret.AlternativeArgument, secret.Secret = new(3), true
	config := bodyInput("Provider configuration (JSON)", "/config", false)
	config.Kind, config.ValueType = "text", "json"
	config.Description = `OIDC requires {"issuer":"https://issuer.example"}; Microsoft accepts {"tenant":"TENANT_ID"}. Leave empty for GitHub/Google or when using --oidc-issuer/--microsoft-tenant. Provide config through only one channel.`
	return []pluginsdk.InputSpec{kind, name, id, secret, config}
}

func regionChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud regions list", "id", "No regions are available; create a private region with fctl cloud regions create or ask an organization administrator for access.", "name", "id")
}

func regionInteractionInputs() map[string][]pluginsdk.InputSpec {
	inputs := map[string][]pluginsdk.InputSpec{"cloud regions create": {nameInput("Private region name")}}
	for _, action := range []string{"show", "delete", "versions"} {
		inputs["cloud regions "+action] = []pluginsdk.InputSpec{argumentChoice("Region", 0, regionChoices())}
	}
	return inputs
}

func stackChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud stack list", "id", "No stacks are available; create one with fctl cloud stack create.", "name", "id")
}

func stackContextInput(index int, all bool) pluginsdk.InputSpec {
	source := stackChoices()
	if all {
		source.Flags = map[string]string{"all": "true"}
	}
	return pluginsdk.InputSpec{Title: "Cloud stack", Kind: "select", Context: "stack", Required: true, AlternativeArgument: new(index), Source: source}
}

func stackActionContextInput(action string) pluginsdk.InputSpec {
	input := stackContextInput(0, true)
	switch action {
	case "upgrade":
		input.Source.MatchFields = map[string][]string{"state": {"ACTIVE"}, "status": {"READY"}}
		input.Source.EmptyMessage = "No active, ready stacks are available to upgrade."
	case "restore":
		input.Source.MatchFields = map[string][]string{"state": {"DELETED"}}
		input.Source.EmptyMessage = "No deleted stacks are available to restore."
	case "enable":
		input.Source.MatchFields = map[string][]string{"state": {"DISABLED"}}
		input.Source.EmptyMessage = "No disabled stacks are available to enable."
	case "disable":
		input.Source.MatchFields = map[string][]string{"state": {"ACTIVE"}}
		input.Source.EmptyMessage = "No active stacks are available to disable."
	case "delete", "update":
		input.Source.MatchFields = map[string][]string{"state": {"ACTIVE", "DISABLED"}}
		input.Source.EmptyMessage = "No active or disabled stacks are available to " + action + "."
	}
	return input
}

func stackInteractionInputs() map[string][]pluginsdk.InputSpec {
	version := payloadFlagChoice("Stack catalog version", "version", cloudChoices("cloud regions versions", "name", "This region has no catalog versions; select another region or ask an organization administrator.", "name"))
	version.Source.Args = []string{"$region"}
	version.Source.PreferredPrefix = "v4."
	version.Source.ExcludeTrueFields = []string{"deprecated"}
	// A default is a suggested choice, never permission to skip the catalog or
	// silently substitute another version when v4.0 is absent (e.g. v4.0-beta).
	version.Default = stackDefaultVersion
	version.Description = "Choose an exact version offered by this region."
	inputs := map[string][]pluginsdk.InputSpec{
		"cloud stack create": {nameInput("Stack name"), payloadFlagChoice("Region", "region", regionChoices()), version},
	}
	for _, action := range []string{"show", "update", "delete", "disable", "enable", "restore", "upgrade", "history", "info", "version"} {
		inputs["cloud stack "+action] = []pluginsdk.InputSpec{stackActionContextInput(action)}
	}
	inputs["cloud stack show"][0].AlternativeFlag = "name"
	name := payloadFlagInput("New stack name", "name")
	name.Description = "Enter a new stack name, or use --data to update metadata without renaming."
	inputs["cloud stack update"] = append(inputs["cloud stack update"], name)
	upgrade := payloadFlagInput("Exact target catalog version", "version")
	upgrade.AlternativeArgument = new(1)
	upgrade.Description = "Enter an exact catalog version from fctl cloud regions versions REGION for this stack's region. The plugin validates it and never falls back to another version."
	inputs["cloud stack upgrade"] = append(inputs["cloud stack upgrade"], upgrade)
	inputs["cloud stack modules list"] = []pluginsdk.InputSpec{stackContextInput(0, true)}
	inputs["cloud stack modules enable"] = []pluginsdk.InputSpec{stackContextInput(1, false), argumentInput("Module name to enable", 0)}
	moduleSource := cloudChoices("cloud stack modules list", "name", "No modules are enabled on this stack; use fctl cloud stack modules enable MODULE.", "name")
	moduleSource.Args = []string{"$stack"}
	inputs["cloud stack modules disable"] = []pluginsdk.InputSpec{stackContextInput(1, false), argumentChoice("Enabled module", 0, moduleSource)}
	inputs["cloud stack users list"] = []pluginsdk.InputSpec{stackContextInput(0, true)}
	policy := payloadFlagChoice("Stack access policy", "policy-id", organizationPolicyChoices())
	policy.ValueType = "number"
	inputs["cloud stack users link"] = []pluginsdk.InputSpec{stackContextInput(1, false), argumentChoice("Organization user", 0, organizationUserChoices()), policy}
	userSource := cloudChoices("cloud stack users list", "id", "No users have access to this stack; link one with fctl cloud stack users link USER --policy-id POLICY.", "name", "email", "id")
	userSource.Args = []string{"$stack"}
	inputs["cloud stack users unlink"] = []pluginsdk.InputSpec{stackContextInput(1, false), argumentChoice("Stack user", 0, userSource)}
	return inputs
}

func appChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud apps list", "id", "No Deploy apps are available; create one with fctl cloud apps create --experimental.", "name", "id")
}

func appManifestChoices() *pluginsdk.ChoiceSource {
	return cloudChoices("cloud apps manifests list", "id", "No manifests are available; create one with fctl cloud apps manifests create --experimental.", "name", "id")
}

func appManifestVersionChoices() *pluginsdk.ChoiceSource {
	source := cloudChoices("cloud apps manifests versions list", "version", "No manifest versions are available; upload one with fctl cloud apps manifests versions push --experimental --manifest-id MANIFEST.", "version")
	source.Flags["manifest-id"] = "$manifest-id"
	return source
}

func appManifestContentInput() pluginsdk.InputSpec {
	input := flagInput("Manifest YAML file", "path", true)
	input.AlternativeFlag = "data"
	input.Description = "Path to a local YAML file; alternatively supply the manifest through --data."
	return input
}

func appInteractionInputs() map[string][]pluginsdk.InputSpec {
	inputs := map[string][]pluginsdk.InputSpec{
		"cloud apps create":          {payloadFlagInput("App name", "name")},
		"cloud apps bind-manifest":   {flagChoice("App", "app-id", appChoices()), payloadFlagChoice("Manifest", "manifest-id", appManifestChoices())},
		"cloud apps unbind-manifest": {flagChoice("App", "app-id", appChoices())},
		// Upload names are query parameters, not fields in the manifest body.
		"cloud apps manifests create":        {flagInput("Manifest name", "name", true), appManifestContentInput()},
		"cloud apps manifests versions push": {flagChoice("Manifest", "manifest-id", appManifestChoices()), appManifestContentInput()},
		"cloud apps manifests versions list": {flagChoice("Manifest", "manifest-id", appManifestChoices())},
		"cloud apps manifests versions show": {flagChoice("Manifest", "manifest-id", appManifestChoices())},
		"cloud apps variables list":          {flagChoice("App", "id", appChoices())},
	}
	for _, action := range []string{"show", "delete"} {
		inputs["cloud apps "+action] = []pluginsdk.InputSpec{flagChoice("App", "id", appChoices())}
	}
	for _, action := range []string{"show", "update", "delete", "download"} {
		inputs["cloud apps manifests "+action] = []pluginsdk.InputSpec{flagChoice("Manifest", "id", appManifestChoices())}
	}
	inputs["cloud apps manifests update"] = append(inputs["cloud apps manifests update"], payloadFlagInput("Manifest name", "name"))
	for _, action := range []string{"show", "logs", "download"} {
		inputs["cloud apps deployments "+action] = []pluginsdk.InputSpec{
			flagChoice("Deployment", "id", cloudChoices("cloud apps deployments list", "id", "No deployments are available; create one with fctl cloud apps deployments create --experimental.", "name", "id", "status")),
		}
	}
	version := payloadFlagChoice("Manifest version", "manifest-version", appManifestVersionChoices())
	version.ValueType = "number"
	inputs["cloud apps deployments create"] = []pluginsdk.InputSpec{payloadFlagChoice("App", "app-id", appChoices()), payloadFlagChoice("Manifest", "manifest-id", appManifestChoices()), version}
	secret := payloadFlagInput("Variable value", "value")
	secret.Secret = true
	inputs["cloud apps variables create"] = []pluginsdk.InputSpec{flagChoice("App", "id", appChoices()), payloadFlagInput("Variable key", "key"), secret}
	variables := cloudChoices("cloud apps variables list", "id", "No variables are available for this app; create one with fctl cloud apps variables create --experimental.", "key", "id")
	variables.Flags["id"] = "$app-id"
	inputs["cloud apps variables delete"] = []pluginsdk.InputSpec{flagChoice("App", "app-id", appChoices()), flagChoice("Variable", "id", variables)}
	return inputs
}
