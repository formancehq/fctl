package plugin_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/plugins/auth"
	cloudplugin "github.com/formancehq/fctl/v4/plugins/cloud"
	"github.com/formancehq/fctl/v4/plugins/ledger"
)

func validationInteractionManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "fixture", Service: "ledger", Version: "test", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "fixture", Target: "stack", Flags: []pluginsdk.FlagSpec{
			{Name: "tenant", Type: "string", Persistent: true}, {Name: "root-only", Type: "string"},
		}, Subcommands: []pluginsdk.CommandSpec{
			{Use: "write [NAME]", Runnable: true, Args: pluginsdk.ArgsSpec{Max: 2}, Flags: []pluginsdk.FlagSpec{
				{Name: "name", Type: "string"}, {Name: "data", Type: "string", Body: true},
				{Name: "enabled", Type: "bool", Default: "false"}, {Name: "limit", Type: "uint32", Default: "10"},
			}, Inputs: []pluginsdk.InputSpec{{Title: "Name", Kind: "input", Flag: "name"}}},
			{Use: "list [FILTER]", Runnable: true, Args: pluginsdk.ArgsSpec{Max: 1}, Flags: []pluginsdk.FlagSpec{
				{Name: "filter", Type: "string"}, {Name: "enabled", Type: "bool", Default: "false"},
				{Name: "after", Type: "string"}, {Name: "page-size", Type: "uint32", Default: "100"},
			}},
			{Use: "group"},
		}},
	}
}

func interactionSource() *pluginsdk.ChoiceSource {
	return &pluginsdk.ChoiceSource{CommandPath: []string{"fixture", "list"}, ValueField: "id", LabelFields: []string{"name", "id"}}
}

func registerInteractionManifest(t *testing.T, manifest pluginsdk.Manifest) error {
	t.Helper()
	registry := &plugin.Registry{}
	err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, func(*http.Client) pluginsdk.Plugin {
		t.Fatal("declaration validation instantiated an executable plugin")
		return nil
	})
	if err != nil && len(registry.List()) != 0 {
		t.Fatal("invalid metadata entered the registry")
	}
	return err
}

func TestInteractionRejectsInvalidInputDeclarations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, message string
		mutate        func(*pluginsdk.InputSpec)
	}{
		{"missing title", "title", func(i *pluginsdk.InputSpec) { i.Title = "  " }},
		{"no binding", "exactly one", func(i *pluginsdk.InputSpec) { i.Flag = "" }},
		{"multiple bindings", "exactly one", func(i *pluginsdk.InputSpec) { i.Argument = new(0) }},
		{"unknown flag", "undeclared flag", func(i *pluginsdk.InputSpec) { i.Flag = "missing" }},
		{"host flag", "undeclared flag", func(i *pluginsdk.InputSpec) { i.Flag = "organization" }},
		{"nonpersistent ancestor flag", "undeclared flag", func(i *pluginsdk.InputSpec) { i.Flag = "root-only" }},
		{"negative argument", "outside command bounds", func(i *pluginsdk.InputSpec) { i.Flag, i.Argument = "", new(-1) }},
		{"argument at max", "outside command bounds", func(i *pluginsdk.InputSpec) { i.Flag, i.Argument = "", new(2) }},
		{"unknown kind", "input kind", func(i *pluginsdk.InputSpec) { i.Kind = "password" }},
		{"unknown value type", "value type", func(i *pluginsdk.InputSpec) { i.ValueType = "float" }},
		{"secret textarea", "secret input", func(i *pluginsdk.InputSpec) { i.Kind, i.Secret = "text", true }},
		{"secret selection", "secret input", func(i *pluginsdk.InputSpec) { i.Kind, i.Secret = "select", true }},
		{"secret default", "secret input", func(i *pluginsdk.InputSpec) { i.Secret, i.Default = true, "synthetic-value" }},
		{"secret numeric value", "secret input", func(i *pluginsdk.InputSpec) { i.Secret, i.ValueType = true, "number" }},
		{"secret boolean flag", "string flag", func(i *pluginsdk.InputSpec) { i.Secret, i.Flag = true, "enabled" }},
		{"unknown context", "input context", func(i *pluginsdk.InputSpec) { i.Flag, i.Context = "", "token" }},
		{"secret host context", "resource context", func(i *pluginsdk.InputSpec) { i.Flag, i.Context, i.Secret = "", "stack", true }},
		{"confirm string", "boolean value type", func(i *pluginsdk.InputSpec) { i.Kind, i.ValueType = "confirm", "string" }},
		{"invalid boolean default", "invalid default", func(i *pluginsdk.InputSpec) { i.Flag, i.ValueType, i.Default = "enabled", "bool", "1" }},
		{"invalid uint32 default", "invalid default", func(i *pluginsdk.InputSpec) { i.Flag, i.Default = "limit", "4294967296" }},
		{"invalid JSON default", "invalid default", func(i *pluginsdk.InputSpec) { i.ValueType, i.Default = "json", "{" }},
		{"quoted numeric default", "invalid default", func(i *pluginsdk.InputSpec) { i.ValueType, i.Default = "number", `"42"` }},
		{"null numeric default", "invalid default", func(i *pluginsdk.InputSpec) { i.ValueType, i.Default = "number", "null" }},
		{"invalid confirm default", "invalid default", func(i *pluginsdk.InputSpec) { i.Kind, i.Default = "confirm", "yes" }},
		{"negative alternative argument", "alternative argument", func(i *pluginsdk.InputSpec) { i.AlternativeArgument = new(-1) }},
		{"alternative argument at max", "alternative argument", func(i *pluginsdk.InputSpec) { i.AlternativeArgument = new(2) }},
		{"missing alternative flag", "alternative", func(i *pluginsdk.InputSpec) { i.AlternativeFlag = "missing" }},
		{"host alternative flag", "alternative", func(i *pluginsdk.InputSpec) { i.AlternativeFlag = "timeout" }},
		{"select without choices", "requires options", func(i *pluginsdk.InputSpec) { i.Kind = "select" }},
		{"options on input", "only select", func(i *pluginsdk.InputSpec) { i.Options = []pluginsdk.InputOption{{Label: "A", Value: "a"}} }},
		{"source on input", "only select", func(i *pluginsdk.InputSpec) { i.Source = interactionSource() }},
		{"source and options", "mutually exclusive", func(i *pluginsdk.InputSpec) {
			i.Kind, i.Source, i.Options = "select", interactionSource(), []pluginsdk.InputOption{{Label: "A", Value: "a"}}
		}},
		{"duplicate option values", "distinct", func(i *pluginsdk.InputSpec) {
			i.Kind, i.Options = "select", []pluginsdk.InputOption{{Label: "A", Value: "a"}, {Label: "Other A", Value: "a"}}
		}},
		{"empty option label", "labels", func(i *pluginsdk.InputSpec) { i.Kind, i.Options = "select", []pluginsdk.InputOption{{Value: "a"}} }},
		{"empty required option", "usable", func(i *pluginsdk.InputSpec) {
			i.Kind, i.Required, i.Options = "select", true, []pluginsdk.InputOption{{Label: "None"}}
		}},
		{"wrong option type", "option value", func(i *pluginsdk.InputSpec) {
			i.Kind, i.ValueType, i.Options = "select", "number", []pluginsdk.InputOption{{Label: "One", Value: "true"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			tc.mutate(&manifest.Root.Subcommands[0].Inputs[0])
			if err := registerInteractionManifest(t, manifest); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("rejection = %v, want %q", err, tc.message)
			}
		})
	}
}

func TestInteractionRejectsInvalidCommandBindings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, message string
		mutate        func(*pluginsdk.Manifest)
	}{
		{"root target", "invalid target", func(m *pluginsdk.Manifest) { m.Root.Target = "tenant" }},
		{"leaf target", "invalid target", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Target = "organizations" }},
		{"group input", "runnable leaf", func(m *pluginsdk.Manifest) { m.Root.Inputs = m.Root.Subcommands[0].Inputs }},
		{"runnable branch input", "runnable leaf", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Subcommands = []pluginsdk.CommandSpec{{Use: "nested"}}
		}},
		{"missing body flag", "body flag", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Flags[1].Body = false
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Name", Kind: "input", BodyPointer: "/name"}}
		}},
		{"duplicate flag binding", "duplicate input binding", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = append(m.Root.Subcommands[0].Inputs, m.Root.Subcommands[0].Inputs[0])
		}},
		{"duplicate argument binding", "duplicate input binding", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "A", Kind: "input", Argument: new(0)}, {Title: "B", Kind: "input", Argument: new(0)}}
		}},
		{"duplicate body binding", "duplicate input binding", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "A", Kind: "input", BodyPointer: "/name"}, {Title: "B", Kind: "input", BodyPointer: "/name"}}
		}},
		{"duplicate context binding", "duplicate input binding", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "A", Kind: "input", Context: "stack"}, {Title: "B", Kind: "input", Context: "stack"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			tc.mutate(&manifest)
			if err := registerInteractionManifest(t, manifest); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("rejection = %v, want %q", err, tc.message)
			}
		})
	}
}

func TestInteractionValidatesRFC6901Pointers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		pointer string
		valid   bool
	}{
		{"/name", true}, {"/a~1b/~0", true}, {"/~01", true}, {"/", true}, {"/a//b", true}, {"/clé", true},
		{"name", false}, {"#/name", false}, {"/name~", false}, {"/~2", false}, {"/~~0", false}, {"/" + string([]byte{0xff}), false},
	} {
		t.Run(tc.pointer, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			manifest.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Body", Kind: "input", BodyPointer: tc.pointer}}
			if err := registerInteractionManifest(t, manifest); (err == nil) != tc.valid {
				t.Fatalf("pointer %q accepted=%v, want %v: %v", tc.pointer, err == nil, tc.valid, err)
			}
		})
	}
}

func TestInteractionRejectsInvalidChoiceSources(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, message string
		mutate        func(*pluginsdk.ChoiceSource)
	}{
		{"other plugin", "same plugin", func(s *pluginsdk.ChoiceSource) { s.CommandPath = []string{"auth", "clients", "list"} }},
		{"missing path", "same plugin", func(s *pluginsdk.ChoiceSource) { s.CommandPath = nil }},
		{"unknown command", "same plugin", func(s *pluginsdk.ChoiceSource) { s.CommandPath = []string{"fixture", "missing"} }},
		{"group command", "runnable command", func(s *pluginsdk.ChoiceSource) { s.CommandPath = []string{"fixture", "group"} }},
		{"too many arguments", "arguments", func(s *pluginsdk.ChoiceSource) { s.Args = []string{"one", "two"} }},
		{"missing value field", "value field", func(s *pluginsdk.ChoiceSource) { s.ValueField = " " }},
		{"empty label field", "label fields", func(s *pluginsdk.ChoiceSource) { s.LabelFields = []string{"id", " "} }},
		{"undeclared source flag", "undeclared flag", func(s *pluginsdk.ChoiceSource) { s.Flags = map[string]string{"missing": "value"} }},
		{"nonpersistent source flag", "undeclared flag", func(s *pluginsdk.ChoiceSource) { s.Flags = map[string]string{"root-only": "value"} }},
		{"invalid source flag value", "choice source flag", func(s *pluginsdk.ChoiceSource) { s.Flags = map[string]string{"enabled": "yes"} }},
		{"unknown input reference", "undeclared input", func(s *pluginsdk.ChoiceSource) { s.Flags = map[string]string{"filter": "$missing"} }},
		{"empty input reference", "undeclared input", func(s *pluginsdk.ChoiceSource) { s.Args = []string{"$"} }},
		{"argument reference too large", "outside command bounds", func(s *pluginsdk.ChoiceSource) { s.Args = []string{"$arg2"} }},
		{"negative argument reference", "outside command bounds", func(s *pluginsdk.ChoiceSource) { s.Args = []string{"$arg-1"} }},
		{"invalid after field", "after field", func(s *pluginsdk.ChoiceSource) { s.AfterField = "id/name" }},
		{"blank after field", "after field", func(s *pluginsdk.ChoiceSource) { s.AfterField = " " }},
		{"empty exclusion field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.ExcludeTrueFields = []string{""} }},
		{"blank exclusion field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.ExcludeTrueFields = []string{" "} }},
		{"nested exclusion field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.ExcludeTrueFields = []string{"catalog.deprecated"} }},
		{"numeric exclusion field prefix", "valid field name", func(s *pluginsdk.ChoiceSource) { s.ExcludeTrueFields = []string{"1deprecated"} }},
		{"duplicate exclusion field", "duplicate excluded field", func(s *pluginsdk.ChoiceSource) {
			s.ExcludeTrueFields = []string{"deprecated", "archived", "deprecated"}
		}},
		{"empty match field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.MatchFields = map[string]string{"": "READY"} }},
		{"blank match field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.MatchFields = map[string]string{" ": "READY"} }},
		{"nested match field", "valid field name", func(s *pluginsdk.ChoiceSource) { s.MatchFields = map[string]string{"stack.status": "READY"} }},
		{"numeric match field prefix", "valid field name", func(s *pluginsdk.ChoiceSource) { s.MatchFields = map[string]string{"1status": "READY"} }},
		{"empty match value", "nonempty value", func(s *pluginsdk.ChoiceSource) { s.MatchFields = map[string]string{"status": ""} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			input := &manifest.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			tc.mutate(input.Source)
			if err := registerInteractionManifest(t, manifest); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("rejection = %v, want %q", err, tc.message)
			}
		})
	}
}

func TestInteractionKeysetPaginationRequiresTypedFlags(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, kind string }{{"after", "bool"}, {"page-size", "string"}, {"after", ""}, {"page-size", ""}} {
		t.Run(tc.name+tc.kind, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			input := &manifest.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			input.Source.AfterField = "id"
			changePaginationFlag(&manifest.Root.Subcommands[1], tc.name, tc.kind)
			if err := registerInteractionManifest(t, manifest); err == nil || !strings.Contains(err.Error(), "after field requires") {
				t.Fatalf("invalid pagination accepted: %v", err)
			}
		})
	}
}

func changePaginationFlag(command *pluginsdk.CommandSpec, name, kind string) {
	for index := range command.Flags {
		flag := &command.Flags[index]
		if flag.Name != name {
			continue
		}
		if kind == "" {
			command.Flags = append(command.Flags[:index], command.Flags[index+1:]...)
			return
		}
		flag.Type = kind
		if kind == "bool" {
			flag.Default = "false"
		}
		return
	}
}

func TestInteractionAllowsSupportedDeclarations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*pluginsdk.Manifest)
	}{
		{"inherited persistent flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Inputs[0].Flag = "tenant" }},
		{"password input", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Inputs[0].Secret = true }},
		{"shared alternative flags", sharedAlternativeInputs},
		{"shared alternative arguments", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{
				{Title: "Name", Kind: "input", Flag: "name", AlternativeArgument: new(0)},
				{Title: "Stack", Kind: "input", Context: "stack", AlternativeArgument: new(0)},
				{Title: "Position", Kind: "input", Argument: new(0)},
			}
		}},
		{"static default outside options", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Name", Kind: "select", Flag: "name", Default: "not-listed", Options: []pluginsdk.InputOption{{Label: "A", Value: "a"}}}}
		}},
		{"optional empty option", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Name", Kind: "select", Flag: "name", Options: []pluginsdk.InputOption{{Label: "None"}}}}
		}},
		{"JSON textarea default", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "JSON", Kind: "text", Flag: "data", ValueType: "json", Default: `{"amount":900719925474099312345}`}}
		}},
		{"exact number default", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Amount", Kind: "input", BodyPointer: "/amount", ValueType: "number", Default: "1e10000"}}
		}},
		{"boolean body default", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Public", Kind: "confirm", BodyPointer: "/public", ValueType: "bool", Default: "false"}}
		}},
		{"dynamic default and preferred prefix", func(m *pluginsdk.Manifest) {
			input := &m.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source, input.Default = "select", interactionSource(), "not-yet-listed"
			input.Source.PreferredPrefix, input.Source.AfterField = "v4.", "id"
			input.Source.Args, input.Source.Flags = []string{"$arg0"}, map[string]string{"filter": "$tenant", "enabled": "true"}
		}},
		{"exclusion field names", func(m *pluginsdk.Manifest) {
			input := &m.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			input.Source.ExcludeTrueFields = []string{"deprecated", "Deprecated", "is_archived", "legacy-status", "_hidden2"}
		}},
		{"matching scalar field values with exclusions", func(m *pluginsdk.Manifest) {
			input := &m.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			input.Source.ExcludeTrueFields = []string{"deprecated"}
			input.Source.MatchFields = map[string]string{"state": "ACTIVE", "status": "READY", "is_enabled": "true", "legacy-status": "Needs review", "_revision2": "7"}
		}},
		{"source mutation guarded at execution", func(m *pluginsdk.Manifest) {
			input := &m.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			input.Source.CommandPath = []string{"fixture", "write"}
		}},
		{"host resource reference", func(m *pluginsdk.Manifest) {
			input := &m.Root.Subcommands[0].Inputs[0]
			input.Kind, input.Source = "select", interactionSource()
			input.Source.Args = []string{"$organization"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := validationInteractionManifest()
			tc.mutate(&manifest)
			if err := registerInteractionManifest(t, manifest); err != nil {
				t.Fatalf("valid metadata rejected: %v", err)
			}
		})
	}
}

func sharedAlternativeInputs(manifest *pluginsdk.Manifest) {
	manifest.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{
		{Title: "Name", Kind: "input", Flag: "name", AlternativeFlag: "data"},
		{Title: "Description", Kind: "input", BodyPointer: "/description", AlternativeFlag: "data"},
		{Title: "Position", Kind: "input", Argument: new(0), AlternativeFlag: "data"},
	}
}

func TestInteractionAllEmbeddedManifestsRegisterAndRootStarts(t *testing.T) {
	t.Parallel()
	registry := &plugin.Registry{}
	for _, factory := range []plugin.Factory{auth.New, cloudplugin.New, ledger.New} {
		if err := registry.Register(t.Context(), factory(nil), factory); err != nil {
			t.Fatalf("embedded manifest registration failed: %v", err)
		}
	}
	if len(registry.List()) != 3 {
		t.Fatal("an embedded manifest was not registered")
	}
	root := cmd.NewRootCommand()
	for _, name := range []string{"auth", "cloud", "ledger"} {
		leaf, _, err := root.Find([]string{name})
		if err != nil || leaf == root {
			t.Fatalf("root command did not attach %s: %v", name, err)
		}
	}
}
