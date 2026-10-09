package cloud

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func interactionCommand(t *testing.T, path string) pluginsdk.CommandSpec {
	t.Helper()
	command, err := pluginsdk.FindCommand(manifest(), strings.Fields(path))
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func TestCloudInteractionTargets(t *testing.T) {
	for path, want := range map[string]string{
		"cloud": "identity", "cloud me": "identity", "cloud me info": "identity",
		"cloud me invitations list": "identity", "cloud organizations list": "identity", "cloud organizations create": "identity",
		"cloud organizations": "organization", "cloud organizations describe": "organization", "cloud organizations users show": "organization",
		"cloud regions": "organization", "cloud regions create": "organization",
		"cloud stack": "organization", "cloud stack create": "organization", "cloud stack modules disable": "organization",
		"cloud apps": "organization", "cloud apps manifests create": "organization",
	} {
		t.Run(path, func(t *testing.T) {
			words := strings.Fields(path)
			var target string
			for i := range words {
				if current := interactionCommand(t, strings.Join(words[:i+1], " ")).Target; current != "" {
					target = current
				}
			}
			if target != want {
				t.Fatalf("inherited target = %q, want %q", target, want)
			}
		})
	}
}

func TestCloudInteractionLeafCoverage(t *testing.T) {
	m := manifest()
	inputs := cloudInteractionInputs()
	t.Logf("interactive fields declared on %d Cloud leaves", len(inputs))
	for path := range inputs {
		if !interactionCommand(t, path).Runnable {
			t.Errorf("inputs attached to non-runnable command %s", path)
		}
	}
	walkCloudInteraction(t, m, m.Root, nil)
	for _, path := range []string{"cloud", "cloud me", "cloud me info", "cloud me invitations list", "cloud organizations list", "cloud stack list"} {
		if inputs := interactionCommand(t, path).Inputs; len(inputs) != 0 {
			t.Errorf("%s should not prompt for fields: %+v", path, inputs)
		}
	}
}

func TestCloudPolicyChoicesUsePolicyIDs(t *testing.T) {
	for _, path := range []string{"cloud organizations users link", "cloud stack users link"} {
		t.Run(path, func(t *testing.T) {
			fields := interactionCommand(t, path).Inputs
			policy := fields[len(fields)-1]
			if policy.Flag != "policy-id" || policy.ValueType != "number" || policy.Source.ValueField != "id" || !slices.Equal(policy.Source.CommandPath, strings.Fields("cloud organizations policies list")) {
				t.Errorf("access policy must come from policy IDs, independently of user choices: %+v", policy)
			}
		})
	}
}

func TestCloudDeployChoiceSourcesExecuteWithOptIn(t *testing.T) {
	cases := []struct {
		command string
		index   int
		flags   map[string]string
		path    string
		data    string
	}{
		{"cloud apps bind-manifest", 0, nil, "/apps", `{"data":[{"id":"app","name":"Books"}]}`},
		{"cloud apps bind-manifest", 1, nil, "/manifests", `{"data":[{"id":"manifest","name":"Production"}]}`},
		{"cloud apps deployments create", 2, map[string]string{"manifest-id": "manifest"}, "/manifests/manifest/versions", `{"data":[{"version":9007199254740993}]}`},
		{"cloud apps variables delete", 1, map[string]string{"id": "app"}, "/apps/app/variables", `{"data":[{"id":"variable","key":"TOKEN"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.command+tc.path, func(t *testing.T) {
			checkCloudDeployChoiceRequest(t, tc.command, tc.index, tc.flags, tc.path, tc.data)
		})
	}
}

func checkCloudDeployChoiceRequest(t *testing.T, command string, index int, flags map[string]string, path, data string) {
	t.Helper()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("discovery request = %s %s, want GET %s", r.Method, r.URL.Path, path)
		}
		cloudTestWrite(t, w, data)
	}))
	defer server.Close()
	source := interactionCommand(t, command).Inputs[index].Source
	if flags == nil {
		flags = make(map[string]string)
	}
	request := pluginsdk.ExecuteRequest{CommandPath: source.CommandPath, Flags: flags, Endpoint: server.URL}
	p := New(server.Client())
	if _, err := p.Execute(t.Context(), request); err == nil || calls != 0 {
		t.Fatal("discovery bypassed the explicit experimental gate")
	}
	request.Flags["experimental"] = "true"
	response, err := p.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	appsEqualJSON(t, response.Data, []byte(data))
	if calls != 1 {
		t.Errorf("expected one read after explicit opt-in, got %d", calls)
	}
}

func walkCloudInteraction(t *testing.T, m pluginsdk.Manifest, command pluginsdk.CommandSpec, parent []string) {
	t.Helper()
	path := append(append([]string{}, parent...), pluginsdk.CommandName(command))
	if !command.Runnable && len(command.Inputs) != 0 {
		t.Errorf("group %s has inherited inputs", strings.Join(path, " "))
	}
	for _, input := range command.Inputs {
		t.Run(strings.Join(path, " ")+"/"+input.Title, func(t *testing.T) {
			checkCloudInputBinding(t, command, input)
			if input.Source != nil {
				checkCloudChoiceSource(t, m, input.Source)
			}
		})
	}
	checkCloudRequiredInputCoverage(t, command, path)
	for _, child := range command.Subcommands {
		walkCloudInteraction(t, m, child, path)
	}
}

func checkCloudRequiredInputCoverage(t *testing.T, command pluginsdk.CommandSpec, path []string) {
	t.Helper()
	for index := range command.Args.Min {
		if !slices.ContainsFunc(command.Inputs, func(input pluginsdk.InputSpec) bool {
			return input.Argument != nil && *input.Argument == index && input.Required
		}) {
			t.Errorf("%s lacks a field for required argument %d", strings.Join(path, " "), index)
		}
	}
	for _, flag := range command.Flags {
		if flag.Required && !slices.ContainsFunc(command.Inputs, func(input pluginsdk.InputSpec) bool { return input.Flag == flag.Name && input.Required }) {
			t.Errorf("%s lacks a field for required flag %s", strings.Join(path, " "), flag.Name)
		}
	}
}

func checkCloudInputBinding(t *testing.T, command pluginsdk.CommandSpec, input pluginsdk.InputSpec) {
	t.Helper()
	bindings := 0
	for _, set := range []bool{input.Flag != "", input.Argument != nil, input.BodyPointer != "", input.Context != ""} {
		if set {
			bindings++
		}
	}
	if bindings != 1 || input.Title == "" {
		t.Fatalf("expected one binding and a title: %+v", input)
	}
	if !slices.Contains([]string{"input", "text", "select", "confirm"}, input.Kind) || !slices.Contains([]string{"", "json", "bool", "number"}, input.ValueType) {
		t.Errorf("invalid field kind/type: %+v", input)
	}
	if input.Flag != "" && !slices.ContainsFunc(command.Flags, func(flag pluginsdk.FlagSpec) bool { return flag.Name == input.Flag }) {
		t.Errorf("field refers to absent leaf flag %q", input.Flag)
	}
	checkCloudInputAlternative(t, command, input)
}

func checkCloudInputAlternative(t *testing.T, command pluginsdk.CommandSpec, input pluginsdk.InputSpec) {
	t.Helper()
	if input.AlternativeFlag != "" && !slices.ContainsFunc(command.Flags, func(flag pluginsdk.FlagSpec) bool { return flag.Name == input.AlternativeFlag }) {
		t.Errorf("alternative refers to absent leaf flag %q", input.AlternativeFlag)
	}
	for _, index := range []*int{input.Argument, input.AlternativeArgument} {
		if index != nil && (*index < 0 || *index >= command.Args.Max) {
			t.Errorf("argument index %d is outside command arity", *index)
		}
	}
	if input.Context != "" && input.Context != "stack" && input.Context != "organization" {
		t.Errorf("unknown context binding %q", input.Context)
	}
	if input.BodyPointer != "" && !strings.HasPrefix(input.BodyPointer, "/") {
		t.Errorf("invalid JSON pointer %q", input.BodyPointer)
	}
	if input.Kind == "select" && input.Source == nil && len(input.Options) == 0 {
		t.Error("select has neither a source nor options")
	}
}

func checkCloudChoiceSource(t *testing.T, m pluginsdk.Manifest, source *pluginsdk.ChoiceSource) {
	t.Helper()
	if len(source.CommandPath) < 3 || source.CommandPath[0] != "cloud" {
		t.Fatalf("source path is not fully rooted: %v", source.CommandPath)
	}
	command, err := pluginsdk.FindCommand(m, source.CommandPath)
	if err != nil {
		t.Fatal(err)
	}
	if !command.Runnable || command.Confirm || !slices.Contains([]string{"list", "versions"}, pluginsdk.CommandName(command)) {
		t.Errorf("source must be a read catalog: %v", source.CommandPath)
	}
	if len(source.Args) < command.Args.Min || len(source.Args) > command.Args.Max {
		t.Errorf("source arguments do not match arity: %+v", source)
	}
	for flag := range source.Flags {
		if !cloudSourceHasFlag(m, source.CommandPath, flag) {
			t.Errorf("source refers to unknown flag %s", flag)
		}
	}
	if len(source.CommandPath) > 1 && source.CommandPath[1] == "apps" && source.Flags["experimental"] != "$experimental" {
		t.Error("Deploy discovery must preserve the user's experimental opt-in")
	}
	if source.ValueField == "" || len(source.LabelFields) == 0 || source.EmptyMessage == "" {
		t.Errorf("source lacks values, labels or empty-list guidance: %+v", source)
	}
}

func cloudSourceHasFlag(m pluginsdk.Manifest, path []string, name string) bool {
	for i := range path {
		command, err := pluginsdk.FindCommand(m, path[:i+1])
		if err != nil {
			return false
		}
		for _, flag := range command.Flags {
			if flag.Name == name && (flag.Persistent || i == len(path)-1) {
				return true
			}
		}
	}
	return false
}

func TestCloudCreationMinimumInputs(t *testing.T) {
	for path, flags := range map[string][]string{
		"cloud organizations create":                            {"name"},
		"cloud regions create":                                  {"name"},
		"cloud organizations policies create":                   {"name"},
		"cloud organizations oauth-clients create":              {"name"},
		"cloud stack create":                                    {"name", "region", "version"},
		"cloud organizations authentication-provider configure": {"type", "name", "provider-client-id", "provider-client-secret"},
		"cloud apps create":                                     {"name"},
		"cloud apps manifests create":                           {"name", "path"},
		"cloud apps manifests versions push":                    {"manifest-id", "path"},
		"cloud apps deployments create":                         {"app-id", "manifest-id", "manifest-version"},
		"cloud apps variables create":                           {"id", "key", "value"},
	} {
		t.Run(path, func(t *testing.T) {
			command := interactionCommand(t, path)
			var required []string
			for _, input := range command.Inputs {
				if input.Required {
					required = append(required, input.Flag)
				}
			}
			if !slices.Equal(required, flags) {
				t.Errorf("required form flags = %v, want %v", required, flags)
			}
		})
	}
	for _, path := range []string{"cloud organizations create", "cloud regions create", "cloud stack create"} {
		name := interactionCommand(t, path).Inputs[0]
		if name.Flag != "name" || name.AlternativeArgument == nil || *name.AlternativeArgument != 0 {
			t.Errorf("%s does not support an already supplied positional name", path)
		}
	}
}

func TestCloudSecretsAndProviderConfiguration(t *testing.T) {
	provider := interactionCommand(t, "cloud organizations authentication-provider configure").Inputs
	for index, input := range provider[:4] {
		if input.AlternativeArgument == nil || *input.AlternativeArgument != index {
			t.Errorf("provider flag %s does not defer to its positional alternative", input.Flag)
		}
	}
	if !provider[3].Secret || provider[3].Default != "" {
		t.Error("provider secret must be masked with no default")
	}
	config := provider[4]
	if config.BodyPointer != "/config" || config.Kind != "text" || config.ValueType != "json" || config.Required || !strings.Contains(config.Description, "issuer") {
		t.Errorf("provider configuration must be a guided optional JSON object: %+v", config)
	}
	variable := interactionCommand(t, "cloud apps variables create").Inputs[2]
	if variable.Flag != "value" || !variable.Secret || variable.Default != "" {
		t.Error("variable value must be masked with no default")
	}
}

func TestCloudStackContextAndDependentCatalogs(t *testing.T) {
	inputs := cloudInteractionInputs()
	for path, fields := range inputs {
		if !strings.HasPrefix(path, "cloud stack ") || path == "cloud stack create" {
			continue
		}
		stack := fields[0]
		if stack.Context != "stack" || stack.Source == nil || !slices.Equal(stack.Source.CommandPath, []string{"cloud", "stack", "list"}) {
			t.Errorf("%s lacks a Cloud stack context picker", path)
		}
	}
	version := interactionCommand(t, "cloud stack create").Inputs[2]
	if version.Flag != "version" || version.Default != "v4.0" || !version.Required || version.Kind != "select" || len(version.Options) != 0 {
		t.Fatalf("version must require a catalog choice: %+v", version)
	}
	if !slices.Equal(version.Source.CommandPath, []string{"cloud", "regions", "versions"}) || !slices.Equal(version.Source.Args, []string{"$region"}) || version.Source.ValueField != "name" || version.Source.PreferredPrefix != "v4." {
		t.Errorf("invalid current region-version source: %+v", version.Source)
	}
	deployment := interactionCommand(t, "cloud apps deployments create").Inputs[2]
	if deployment.ValueType != "number" || deployment.Source.ValueField != "version" || deployment.Source.Flags["manifest-id"] != "$manifest-id" {
		t.Errorf("manifest version must retain its integer and depend on the chosen manifest: %+v", deployment)
	}
	variable := interactionCommand(t, "cloud apps variables delete").Inputs[1]
	if variable.Source.Flags["id"] != "$app-id" || slices.Contains(variable.Source.LabelFields, "value") {
		t.Errorf("variable picker must depend on its app and exclude secrets: %+v", variable)
	}
}

func TestCloudStackCatalogExcludesDeprecatedVersions(t *testing.T) {
	version := interactionCommand(t, "cloud stack create").Inputs[2]
	if !slices.Equal(version.Source.ExcludeTrueFields, []string{"deprecated"}) {
		t.Errorf("version choices must exclude deprecated catalog entries: %+v", version.Source)
	}
}

func TestCloudStackActionEligibility(t *testing.T) {
	for _, tc := range []struct {
		action       string
		eligible     map[string][]string
		emptyMessage string
	}{
		{"upgrade", map[string][]string{"state": {"ACTIVE"}, "status": {"READY"}}, "No active, ready stacks are available to upgrade."},
		{"restore", map[string][]string{"state": {"DELETED"}}, "No deleted stacks are available to restore."},
		{"enable", map[string][]string{"state": {"DISABLED"}}, "No disabled stacks are available to enable."},
		{"disable", map[string][]string{"state": {"ACTIVE"}}, "No active stacks are available to disable."},
		{"delete", map[string][]string{"state": {"ACTIVE", "DISABLED"}}, "No active or disabled stacks are available to delete."},
		{"update", map[string][]string{"state": {"ACTIVE", "DISABLED"}}, "No active or disabled stacks are available to update."},
	} {
		t.Run(tc.action, func(t *testing.T) {
			source := interactionCommand(t, "cloud stack "+tc.action).Inputs[0].Source
			if !maps.EqualFunc(source.MatchFields, tc.eligible, slices.Equal[[]string]) {
				t.Errorf("%s must offer stacks matching %v, got %v", tc.action, tc.eligible, source.MatchFields)
			}
			if source.EmptyMessage != tc.emptyMessage {
				t.Errorf("empty selection must explain the action's eligible state: %q", source.EmptyMessage)
			}
			if !maps.Equal(source.Flags, map[string]string{"all": "true"}) {
				t.Errorf("action eligibility must inspect the full stack catalog: %v", source.Flags)
			}
		})
	}
}

func TestCloudStackReadChoicesRetainDeletedStacks(t *testing.T) {
	for _, action := range []string{"show", "history", "info", "version", "modules list", "users list"} {
		t.Run(action, func(t *testing.T) {
			source := interactionCommand(t, "cloud stack "+action).Inputs[0].Source
			if len(source.MatchFields) != 0 || !maps.Equal(source.Flags, map[string]string{"all": "true"}) {
				t.Errorf("%s must keep deleted stacks available for inspection: %+v", action, source)
			}
		})
	}
}

func TestCloudManifestContentAcceptsExplicitBody(t *testing.T) {
	for _, path := range []string{"cloud apps manifests create", "cloud apps manifests versions push"} {
		command := interactionCommand(t, path)
		content := command.Inputs[1]
		if content.Flag != "path" || content.AlternativeFlag != "data" || !content.Required {
			t.Errorf("%s must skip the file prompt when --data is explicit: %+v", path, content)
		}
		if !slices.ContainsFunc(command.Flags, func(flag pluginsdk.FlagSpec) bool { return flag.Name == content.AlternativeFlag && flag.Body }) {
			t.Errorf("%s content alternative is not a body flag", path)
		}
	}
}

func TestCloudPayloadFieldsAcceptExplicitData(t *testing.T) {
	for path, flags := range map[string][]string{
		"cloud organizations create":                            {"name"},
		"cloud organizations update":                            {"name"},
		"cloud organizations policies create":                   {"name"},
		"cloud organizations policies update":                   {"name"},
		"cloud organizations oauth-clients create":              {"name"},
		"cloud organizations oauth-clients update":              {"name"},
		"cloud organizations users link":                        {"policy-id"},
		"cloud organizations authentication-provider configure": {"type", "name", "provider-client-id", "provider-client-secret"},
		"cloud regions create":                                  {"name"},
		"cloud stack create":                                    {"name", "region", "version"},
		"cloud stack update":                                    {"name"},
		"cloud stack upgrade":                                   {"version"},
		"cloud stack users link":                                {"policy-id"},
		"cloud apps create":                                     {"name"},
		"cloud apps bind-manifest":                              {"manifest-id"},
		"cloud apps manifests update":                           {"name"},
		"cloud apps deployments create":                         {"app-id", "manifest-id", "manifest-version"},
		"cloud apps variables create":                           {"key", "value"},
	} {
		t.Run(path, func(t *testing.T) {
			command := interactionCommand(t, path)
			for _, flag := range flags {
				input := cloudFlagField(t, command, flag)
				if input.AlternativeFlag != "data" {
					t.Errorf("payload field --%s must defer to explicit --data: %+v", flag, input)
				}
			}
			if !slices.ContainsFunc(command.Flags, func(flag pluginsdk.FlagSpec) bool { return flag.Name == "data" && flag.Body }) {
				t.Error("payload alternatives require an existing body flag")
			}
		})
	}
}

func cloudFlagField(t *testing.T, command pluginsdk.CommandSpec, flag string) pluginsdk.InputSpec {
	t.Helper()
	for _, input := range command.Inputs {
		if input.Flag == flag {
			return input
		}
	}
	t.Fatalf("command %s has no field for --%s", command.Use, flag)
	return pluginsdk.InputSpec{}
}

func TestCloudResourceTargetsDoNotDeferToData(t *testing.T) {
	cases := []struct {
		path  string
		index int
	}{
		{"cloud organizations update", 0},
		{"cloud organizations policies update", 0},
		{"cloud organizations oauth-clients update", 0},
		{"cloud organizations users link", 0},
		{"cloud stack update", 0},
		{"cloud stack upgrade", 0},
		{"cloud stack users link", 0},
		{"cloud stack users link", 1},
		{"cloud apps bind-manifest", 0},
		{"cloud apps manifests update", 0},
		{"cloud apps manifests versions push", 0},
		{"cloud apps variables create", 0},
		{"cloud apps variables delete", 0},
		{"cloud apps variables delete", 1},
		// A manifest upload body also does not supply the name query parameter.
		{"cloud apps manifests create", 0},
	}
	for _, tc := range cases {
		input := interactionCommand(t, tc.path).Inputs[tc.index]
		if input.AlternativeFlag == "data" || !input.Required {
			t.Errorf("%s field %s must still resolve its target with --data: %+v", tc.path, input.Title, input)
		}
	}
}

func TestCloudUpdateFormsRequireExplicitNameOrData(t *testing.T) {
	for _, path := range []string{"cloud organizations update", "cloud organizations policies update", "cloud organizations oauth-clients update", "cloud stack update", "cloud apps manifests update"} {
		name := cloudFlagField(t, interactionCommand(t, path), "name")
		if !name.Required || name.AlternativeFlag != "data" || name.Default != "" || name.Source != nil {
			t.Errorf("%s must require an explicit new name or JSON update, without a default mutation: %+v", path, name)
		}
	}
}

func TestCloudStackShowUsesExplicitNameInsteadOfPicker(t *testing.T) {
	stack := interactionCommand(t, "cloud stack show").Inputs[0]
	if stack.Context != "stack" || stack.AlternativeFlag != "name" || stack.AlternativeArgument == nil || *stack.AlternativeArgument != 0 {
		t.Errorf("show must permit --name and the existing positional stack ID: %+v", stack)
	}
}

func TestCloudExplicitJSONStackUpdatePreservesName(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.URL.Path != "/organizations/org/stacks/stack" {
			t.Errorf("update targeted %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			cloudTestWrite(t, w, `{"data":{"id":"stack","name":"existing","metadata":{"owner":"previous"}}}`)
		case http.MethodPut:
			var body json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			appsEqualJSON(t, body, []byte(`{"name":"existing","metadata":{"owner":"accepted"}}`))
			cloudTestWrite(t, w, `{"data":{"id":"stack"}}`)
		default:
			t.Errorf("unexpected update method %s", r.Method)
		}
	}))
	defer server.Close()
	request := pluginsdk.ExecuteRequest{
		CommandPath: strings.Fields("cloud stack update"),
		Flags:       map[string]string{"data": `{"metadata":{"owner":"accepted"}}`}, ChangedFlags: map[string]bool{"data": true},
		Body: json.RawMessage(`{"metadata":{"owner":"accepted"}}`), Context: map[string]string{"organization": "org", "stack": "stack"}, Endpoint: server.URL,
	}
	if _, err := New(server.Client()).Execute(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(methods, []string{http.MethodGet, http.MethodPut}) {
		t.Errorf("update should read then preserve the omitted name: %v", methods)
	}
}

func TestCloudInteractionPreservesRequestContract(t *testing.T) {
	decorated := manifest()
	original := decorated
	original.Root = pluginsdk.CommandSpec{Use: "cloud", Subcommands: []pluginsdk.CommandSpec{meManifest(), organizationsManifest(), regionsManifest(), stackManifest(), appsManifest()}}
	for _, request := range []pluginsdk.ExecuteRequest{
		{CommandPath: strings.Fields("cloud organizations create"), Args: []string{"Books"}},
		{CommandPath: strings.Fields("cloud organizations create"), Body: json.RawMessage(`{"name":"Books","defaultPolicyID":9007199254740993}`)},
		{CommandPath: strings.Fields("cloud stack create"), Flags: map[string]string{"name": "Books", "region": "eu", "version": "v4.0-beta"}, ChangedFlags: map[string]bool{"version": true}},
		{CommandPath: strings.Fields("cloud apps variables create"), Flags: map[string]string{"experimental": "true", "id": "app"}, Body: json.RawMessage(`{"variable":{"key":"token","value":"synthetic-secret"}}`)},
		{CommandPath: strings.Fields("cloud organizations policies delete"), Args: []string{"4"}},
		{CommandPath: strings.Fields("cloud regions show")},
	} {
		before, beforeErr := pluginsdk.NormalizeRequest(original, request)
		after, afterErr := pluginsdk.NormalizeRequest(decorated, request)
		if !reflect.DeepEqual(before, after) || errorString(beforeErr) != errorString(afterErr) {
			t.Errorf("metadata changed normalization of %v: %v / %v", request.CommandPath, beforeErr, afterErr)
		}
	}
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func TestCloudInteractionManifestIsolation(t *testing.T) {
	first := manifest()
	before, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"cloud stack create", "cloud stack delete", "cloud apps variables delete", "cloud organizations authentication-provider configure"} {
		command, err := pluginsdk.FindCommand(first, strings.Fields(path))
		if err != nil {
			t.Fatal(err)
		}
		for _, input := range command.Inputs {
			mutateCloudInputMetadata(input)
		}
	}
	after, err := json.Marshal(manifest())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("a mutated manifest leaked interactive metadata into the next call")
	}
}

func mutateCloudInputMetadata(input pluginsdk.InputSpec) {
	if input.AlternativeArgument != nil {
		*input.AlternativeArgument = 99
	}
	if input.Source != nil {
		input.Source.CommandPath[0] = "changed"
		if len(input.Source.ExcludeTrueFields) > 0 {
			input.Source.ExcludeTrueFields[0] = "changed"
		}
		for key := range input.Source.Flags {
			input.Source.Flags[key] = "changed"
		}
		for _, values := range input.Source.MatchFields {
			for index := range values {
				values[index] = "changed"
			}
		}
	}
	if len(input.Options) > 0 {
		input.Options[0].Value = "changed"
	}
}

func TestCloudInteractiveCatalogVersionSubmission(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		writeCloudCatalogVersionFixture(t, w, r)
	}))
	defer server.Close()
	p := New(server.Client())
	version := interactionCommand(t, "cloud stack create").Inputs[2]
	request := pluginsdk.ExecuteRequest{CommandPath: version.Source.CommandPath, Args: []string{"eu"}, Context: map[string]string{"organization": "org"}, Endpoint: server.URL}
	response, err := p.Execute(t.Context(), request)
	if err != nil || !strings.Contains(string(response.Data), "v4.0-beta") {
		t.Fatalf("resolve version catalog: %s / %v", response.Data, err)
	}
	if !slices.Equal(methods, []string{http.MethodGet}) {
		t.Fatalf("reading choices performed writes: %v", methods)
	}
	request.CommandPath = strings.Fields("cloud stack create")
	request.Args = []string{"Books"}
	request.Flags = map[string]string{"region": "eu", "version": "v4.0-beta", "no-wait": "true"}
	request.ChangedFlags = map[string]bool{"version": true}
	if _, err := p.Execute(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(methods, []string{http.MethodGet, http.MethodGet, http.MethodPost}) {
		t.Errorf("catalog lookup should precede the single authorized creation: %v", methods)
	}
}

func writeCloudCatalogVersionFixture(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	switch r.URL.Path {
	case "/organizations/org/regions/eu/versions":
		if r.Method != http.MethodGet {
			t.Errorf("catalog source mutated a resource: %s", r.Method)
		}
		cloudTestWrite(t, w, `{"data":[{"name":"v4.0-beta"}]}`)
	case "/organizations/org/stacks":
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if string(body["version"]) != `"v4.0-beta"` {
			t.Errorf("chosen catalog version was replaced: %s", body["version"])
		}
		cloudTestWrite(t, w, `{"data":{"id":"created","organizationId":"org"}}`)
	default:
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}
}
