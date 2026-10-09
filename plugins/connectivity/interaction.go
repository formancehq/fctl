package connectivity

import "github.com/formancehq/fctl/pkg/pluginsdk"

func connectorInput() pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: "Connector", Kind: "select", Argument: new(0), ValueType: "string", Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"connectivity", "connectors", "list"}, ValueField: "metadata.name", LabelFields: []string{"spec.displayName"}, EmptyMessage: "No published connectors are available"}}
}

func instanceInput() pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: "Connector instance", Kind: "select", Argument: new(0), ValueType: "string", Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"connectivity", "instances", "list"}, ValueField: "metadata.name", LabelFields: []string{"spec.connector", "spec.ledger", "status.phase"}, EmptyMessage: "No connector instances are available; create one first"}}
}

func versionInput() pluginsdk.InputSpec {
	return pluginsdk.InputSpec{Title: "Connector version", Kind: "select", Argument: new(1), ValueType: "string", Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"connectivity", "connectors", "versions", "list"}, Args: []string{"$arg0"}, ValueField: "version", LabelFields: []string{"phase"}, EmptyMessage: "This connector has no published versions; specify a channel alias explicitly if needed"}}
}

func createInputs() []pluginsdk.InputSpec {
	connector := connectorInput()
	connector.Argument, connector.BodyPointer = nil, "/spec/connector"
	return []pluginsdk.InputSpec{
		{Title: "Instance name", Description: "Lowercase letters, digits and hyphens", Kind: "input", BodyPointer: "/name", ValueType: "string", Required: true},
		connector,
		{Title: "Ledger", Description: "Destination ledger name", Kind: "input", BodyPointer: "/spec/ledger", ValueType: "string", Required: true},
		{Title: "Version pin", Description: "Optional exact version; leave blank to select a channel", Kind: "input", BodyPointer: "/spec/version", ValueType: "string"},
		{Title: "Channel", Description: "Used when no version is pinned", Kind: "select", BodyPointer: "/spec/channel", ValueType: "string", Default: "stable", Options: channels()},
		{Title: "Configuration", Description: "JSON object with env and files matching the connector version's configSchema; use secretRef for sensitive values", Kind: "text", BodyPointer: "/spec/config", ValueType: "json"},
		{Title: "Start sequence", Description: "Optional nonnegative int64 ingestion sequence", Kind: "input", BodyPointer: "/spec/startSequence", ValueType: "number"},
		{Title: "Poll interval", Description: "Optional Go duration, e.g. 30s", Kind: "input", BodyPointer: "/spec/pollInterval", ValueType: "string"},
		{Title: "Suspend ingestion", Kind: "confirm", BodyPointer: "/spec/suspend", ValueType: "bool", Default: "false"},
	}
}

func channels() []pluginsdk.InputOption {
	return []pluginsdk.InputOption{{Label: "Stable", Value: "stable"}, {Label: "Release candidate", Value: "rc"}, {Label: "Beta", Value: "beta"}, {Label: "Alpha", Value: "alpha"}}
}

func patchInputs() []pluginsdk.InputSpec {
	// No defaults: omitted fields must preserve the existing spec. Explicit
	// --data also supports null deletion and every API configuration field.
	return []pluginsdk.InputSpec{
		instanceInput(),
		{Title: "Spec patch", Description: "JSON object applied directly to spec, e.g. {\"suspend\":true}; use null to remove a field", Kind: "text", Flag: "data", ValueType: "string", Required: true},
	}
}
