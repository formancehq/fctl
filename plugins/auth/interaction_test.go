package auth

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

func TestInteractionManifestContract(t *testing.T) {
	t.Parallel()
	m := manifest()
	if m.Root.Target != "stack" {
		t.Fatal("Auth must target stack context")
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pluginsdk.Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, decoded) {
		t.Fatal("input metadata does not survive serialization")
	}
	walkInputs(t, m, m.Root, nil)
}

func walkInputs(t *testing.T, m pluginsdk.Manifest, command pluginsdk.CommandSpec, parent []string) {
	t.Helper()
	path := append(append([]string{}, parent...), pluginsdk.CommandName(command))
	if !command.Runnable && len(command.Inputs) != 0 {
		t.Fatalf("%s has inherited inputs", path)
	}
	for _, input := range command.Inputs {
		checkInputBinding(t, command, input)
		if input.Source != nil {
			source, err := pluginsdk.FindCommand(m, input.Source.CommandPath)
			if err != nil || !source.Runnable || source.Confirm {
				t.Fatalf("%s has invalid source: %#v, %v", path, input.Source, err)
			}
			if len(input.Source.Args) != source.Args.Min || input.Source.ValueField == "" || input.Source.EmptyMessage == "" {
				t.Fatalf("incomplete source: %#v", input.Source)
			}
		}
	}
	for _, child := range command.Subcommands {
		walkInputs(t, m, child, path)
	}
}

func checkInputBinding(t *testing.T, command pluginsdk.CommandSpec, input pluginsdk.InputSpec) {
	t.Helper()
	bindings := 0
	for _, bound := range []bool{input.Flag != "", input.Argument != nil, input.BodyPointer != "", input.Context != ""} {
		if bound {
			bindings++
		}
	}
	if bindings != 1 || input.Title == "" || !slices.Contains([]string{"input", "text", "select", "confirm"}, input.Kind) || !slices.Contains([]string{"", "string", "json", "bool", "number"}, input.ValueType) {
		t.Fatalf("invalid input: %#v", input)
	}
	if input.Argument != nil && (*input.Argument < 0 || *input.Argument >= command.Args.Max) {
		t.Fatalf("argument binding out of range: %#v", input)
	}
	if input.BodyPointer != "" {
		checkBodyPointer(t, input.BodyPointer)
	}
}

func checkBodyPointer(t *testing.T, pointer string) {
	t.Helper()
	if !strings.HasPrefix(pointer, "/") {
		t.Fatalf("invalid JSON pointer %q", pointer)
	}
	for i := 0; i < len(pointer); i++ {
		if pointer[i] == '~' {
			i++
			if i >= len(pointer) || (pointer[i] != '0' && pointer[i] != '1') {
				t.Fatalf("invalid JSON pointer escape %q", pointer)
			}
		}
	}
}

func TestAuthCreateBodyInputsMatchSDK(t *testing.T) {
	t.Parallel()
	command, err := pluginsdk.FindCommand(manifest(), strings.Fields("auth clients create"))
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"/name": `"automation"`, "/description": `"Service client"`, "/public": `false`, "/trusted": `false`,
		"/scopes": `["ledger:read"]`, "/redirectUris": `["https://example.test/callback"]`,
		"/postLogoutRedirectUris": `["https://example.test/logout"]`, "/metadata": `{"purpose":"tests"}`,
	}
	body := make(map[string]json.RawMessage)
	for _, input := range command.Inputs {
		value, ok := values[input.BodyPointer]
		if !ok {
			t.Fatalf("unknown SDK field %q", input.BodyPointer)
		}
		body[strings.TrimPrefix(input.BodyPointer, "/")] = json.RawMessage(value)
	}
	if len(body) != len(values) || !command.Inputs[0].Required {
		t.Fatal("incomplete client creation form")
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	_, err = execute(t, request("clients create", nil, string(data), false), func(req *http.Request) (*http.Response, error) {
		assertJSON(t, readRequest(t, req), data)
		return reply(http.StatusCreated, `{"data":{"id":"c1","name":"automation"}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthUpdateInputsPreserveOmittedOptions(t *testing.T) {
	t.Parallel()
	command, err := pluginsdk.FindCommand(manifest(), strings.Fields("auth clients update"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range command.Inputs {
		if input.BodyPointer == "" {
			continue
		}
		if !slices.Contains([]string{"/name", "/description"}, input.BodyPointer) || input.Default != "" || input.Kind == "confirm" {
			t.Fatalf("update injects an omitted option: %#v", input)
		}
		if input.BodyPointer == "/description" && input.Required {
			t.Fatal("description must be optional")
		}
	}
	current := `{"data":{"name":"old","description":"preserved","public":true,"trusted":true,"scopes":["ledger:read"],"metadata":{"purpose":"original"}}}`
	patch := `{"name":"renamed"}`
	merged, err := mergeOptions(json.RawMessage(current), json.RawMessage(patch))
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, merged, []byte(`{"name":"renamed","description":"preserved","public":true,"trusted":true,"scopes":["ledger:read"],"metadata":{"purpose":"original"}}`))
}

func TestAuthChoiceSourcesUseInjectedReadClient(t *testing.T) {
	t.Parallel()
	cases := []struct {
		command, field, source, response, value string
		argument                                int
	}{
		{"clients show", "id", "clients list", `{"data":[{"id":"c1","name":"Automation"}]}`, "c1", 0},
		{"clients delete", "id", "clients list", `{"data":[{"id":"c1","name":"Automation"}]}`, "c1", 0},
		{"clients update", "id", "clients list", `{"data":[{"id":"c1","name":"Automation"}]}`, "c1", 0},
		{"clients secrets list", "id", "clients list", `{"data":[{"id":"c1","name":"Automation"}]}`, "c1", 0},
		{"clients secrets create", "id", "clients list", `{"data":[{"id":"c1","name":"Automation"}]}`, "c1", 0},
		{"clients secrets delete", "id", "clients secrets list", `{"data":{"id":"c1","name":"Automation","secrets":[{"id":"s1","name":"CI"}]}}`, "s1", 1},
		{"users show", "id", "users list", `{"data":[{"id":"u1","subject":"sub","email":"user@example.test"}]}`, "u1", 0},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			checkAuthChoiceSource(t, tc.command, tc.argument, tc.source, tc.response, tc.field, tc.value)
		})
	}
}

func checkAuthChoiceSource(t *testing.T, command string, argument int, sourcePath, response, field, value string) {
	t.Helper()
	spec, err := pluginsdk.FindCommand(manifest(), strings.Fields("auth "+command))
	if err != nil {
		t.Fatal(err)
	}
	index := slices.IndexFunc(spec.Inputs, func(input pluginsdk.InputSpec) bool { return input.Argument != nil && *input.Argument == argument })
	if index < 0 || spec.Inputs[index].Source == nil {
		t.Fatal("missing selector")
	}
	source := spec.Inputs[index].Source
	if strings.Join(source.CommandPath, " ") != "auth "+sourcePath || source.ValueField != field {
		t.Fatalf("source = %#v", source)
	}
	args := slices.Clone(source.Args)
	for i, arg := range args {
		if arg != "$arg0" {
			t.Fatalf("unexpected substitution %q", arg)
		}
		args[i] = "c1"
	}
	calls := 0
	result, err := execute(t, pluginsdk.ExecuteRequest{CommandPath: source.CommandPath, Args: args}, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet {
			t.Fatalf("choice source attempted %s", req.Method)
		}
		return reply(http.StatusOK, response), nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("source execution: calls=%d, error=%v", calls, err)
	}
	rows := authChoiceRows(t, result.Data)
	if len(rows) != 1 || rows[0][field] != value {
		t.Fatalf("selector cannot read %s: %s", field, result.Data)
	}
}

func authChoiceRows(t *testing.T, data json.RawMessage) []map[string]string {
	t.Helper()
	var envelope struct {
		Data []map[string]string `json:"data"`
	}
	if strings.HasPrefix(string(data), "[") {
		decode(t, data, &envelope.Data)
	} else {
		decode(t, data, &envelope)
	}
	return envelope.Data
}
