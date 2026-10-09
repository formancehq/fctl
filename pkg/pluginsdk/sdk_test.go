package pluginsdk_test

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func contractManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{
		Name: "example", Version: "1.2.3", Service: "ledger", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{
			Use: "example", Short: "Example plugin", Long: "Contract fixture", Example: "example records get 9007199254740993",
			Flags: []pluginsdk.FlagSpec{
				{Name: "region", Type: "string", Default: "global", Persistent: true},
				{Name: "root-only", Type: "string", Default: "root"},
			},
			Subcommands: []pluginsdk.CommandSpec{{
				Use: "records", Short: "Manage records",
				Flags: []pluginsdk.FlagSpec{{Name: "tenant", Type: "string", Default: "default-tenant", Persistent: true}, {Name: "parent-only", Type: "string", Default: "parent"}},
				Subcommands: []pluginsdk.CommandSpec{{
					Use: "get ID [SECOND_ID]", Short: "Read records", Runnable: true, Args: pluginsdk.ArgsSpec{Min: 1, Max: 2},
					Flags: []pluginsdk.FlagSpec{
						{Name: "format", Shorthand: "f", Type: "string", Default: "json", Usage: "Output format"},
						{Name: "enabled", Type: "bool", Default: "true"},
						{Name: "limit", Type: "uint32", Default: "100"},
						{Name: "required", Type: "string", Required: true},
						{Name: "body", Type: "string", Body: true},
					},
				}},
			}},
		},
	}
}

func contractRequest() pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{CommandPath: []string{"example", "records", "get"}, Args: []string{"9007199254740993"}, Endpoint: "https://stack.example/api/ledger", Flags: map[string]string{"required": "present"}}
}

func TestFindCommand(t *testing.T) {
	manifest := contractManifest()
	tests := []struct {
		name     string
		path     []string
		use      string
		runnable bool
	}{
		{"root", []string{"example"}, "example", false},
		{"parent", []string{"example", "records"}, "records", false},
		{"leaf strips usage arguments", []string{"example", "records", "get"}, "get ID [SECOND_ID]", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, err := pluginsdk.FindCommand(manifest, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if command.Use != test.use || command.Runnable != test.runnable {
				t.Fatalf("unexpected command: %+v", command)
			}
			if !reflect.DeepEqual(command, testCommand(manifest, test.path)) {
				t.Fatal("FindCommand dropped command metadata")
			}
		})
	}
}

func testCommand(manifest pluginsdk.Manifest, path []string) pluginsdk.CommandSpec {
	command := manifest.Root
	for range len(path) - 1 {
		command = command.Subcommands[0]
	}
	return command
}

func TestFindCommandRejectsUnknownRoutes(t *testing.T) {
	for _, path := range [][]string{nil, {}, {"other"}, {"example", "missing"}, {"example", "records", "missing"}, {"records", "get"}, {"example", "records", "get", "extra"}, {"example records get"}} {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			if _, err := pluginsdk.FindCommand(contractManifest(), path); err == nil {
				t.Fatal("accepted an unknown route")
			}
			request := contractRequest()
			request.CommandPath = path
			if _, err := pluginsdk.NormalizeRequest(contractManifest(), request); err == nil {
				t.Fatal("normalization accepted an unknown route")
			}
		})
	}
}

func TestFindCommandHandlesBlankUsage(t *testing.T) {
	manifest := contractManifest()
	manifest.Root.Use = " \t"
	if _, err := pluginsdk.FindCommand(manifest, []string{"example"}); err == nil {
		t.Fatal("accepted blank root usage")
	}
	manifest = contractManifest()
	manifest.Root.Subcommands = append([]pluginsdk.CommandSpec{{Use: " \t"}}, manifest.Root.Subcommands...)
	if _, err := pluginsdk.FindCommand(manifest, contractRequest().CommandPath); err != nil {
		t.Fatalf("blank sibling hid a valid route: %v", err)
	}
}

func TestNormalizeDefaultsAndInheritedFlags(t *testing.T) {
	request := contractRequest()
	normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"region": "global", "tenant": "default-tenant", "format": "json", "enabled": "true", "limit": "100", "required": "present", "body": ""}
	if !maps.Equal(normalized.Flags, want) {
		t.Fatalf("defaults/inheritance mismatch: %v", normalized.Flags)
	}
	if normalized.Endpoint != request.Endpoint || !reflect.DeepEqual(normalized.Args, request.Args) || !reflect.DeepEqual(normalized.CommandPath, request.CommandPath) {
		t.Fatal("normalization changed routing or arguments")
	}
	for _, name := range []string{"root-only", "parent-only"} {
		request.Flags[name] = "explicit"
		if _, err := pluginsdk.NormalizeRequest(contractManifest(), request); err == nil {
			t.Fatalf("inherited non-persistent flag %s", name)
		}
		delete(request.Flags, name)
	}
}

func TestNormalizeExplicitValuesAndChangedFlags(t *testing.T) {
	request := contractRequest()
	request.Flags["enabled"] = "false"
	request.Flags["region"] = "eu"
	request.Flags["format"] = ""
	request.Flags["limit"] = "0"
	request.ChangedFlags = map[string]bool{"enabled": true, "region": true, "format": true, "limit": true, "tenant": false}
	normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"enabled", "region", "format", "limit"} {
		if normalized.Flags[name] != request.Flags[name] {
			t.Fatalf("default replaced explicit %s", name)
		}
	}
	if !maps.Equal(normalized.ChangedFlags, request.ChangedFlags) {
		t.Fatal("ChangedFlags did not preserve explicit false/unchanged markers")
	}
	if _, exists := normalized.ChangedFlags["body"]; exists {
		t.Fatal("normalization marked a default as changed")
	}
}

func TestNormalizeDoesNotMutateCaller(t *testing.T) {
	request := contractRequest()
	request.ChangedFlags = map[string]bool{"required": true}
	request.Body = json.RawMessage(`{"id":9007199254740993}`)
	before := jsonClone(t, request)
	normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatal("normalization mutated caller state")
	}
	normalized.Flags["required"] = "changed-after-normalize"
	if request.Flags["required"] != "present" {
		t.Fatal("normalized flags share the caller map")
	}
	request.Flags["required"] = "changed-after-call"
	if normalized.Flags["required"] != "changed-after-normalize" {
		t.Fatal("caller flags share the normalized map")
	}
}

func TestNormalizeContextIsTransportNeutralAndIsolated(t *testing.T) {
	t.Parallel()
	request := contractRequest()
	request.Context = map[string]string{"organization": "org-1", "stack": "stack-1", "organizationIDs": `["org-1","org-2"]`}
	before := jsonClone(t, request)
	normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request, before) || !maps.Equal(normalized.Context, before.Context) {
		t.Fatal("normalization changed caller state or host metadata")
	}
	if restored := jsonClone(t, normalized); !maps.Equal(restored.Context, normalized.Context) {
		t.Fatal("protocol serialization lost context metadata")
	}
	normalized.Context["organization"] = "plugin-change"
	if request.Context["organization"] != "org-1" {
		t.Fatal("plugin metadata mutation reached the caller")
	}
	request.Context["stack"] = "host-change"
	if normalized.Context["stack"] != "stack-1" {
		t.Fatal("caller metadata mutation reached the normalized request")
	}
}

func TestNormalizeContextIsOptional(t *testing.T) {
	t.Parallel()
	request := contractRequest()
	normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil || normalized.Context != nil || request.Context != nil {
		t.Fatalf("nil context changed: normalized=%v request=%v err=%v", normalized.Context, request.Context, err)
	}
	request.Context = map[string]string{}
	normalized, err = pluginsdk.NormalizeRequest(contractManifest(), request)
	if err != nil || normalized.Context == nil {
		t.Fatalf("explicit empty context changed: %v, %v", normalized.Context, err)
	}
	normalized.Context["region"] = "eu"
	if len(request.Context) != 0 {
		t.Fatal("empty context maps share storage")
	}
}

func TestNormalizeFailureDoesNotMutateCaller(t *testing.T) {
	request := contractRequest()
	request.Flags["enabled"] = "not-a-bool"
	request.ChangedFlags = map[string]bool{"enabled": true}
	before := jsonClone(t, request)
	if _, err := pluginsdk.NormalizeRequest(contractManifest(), request); err == nil {
		t.Fatal("accepted invalid bool")
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatal("failed normalization mutated caller flags")
	}
}

func TestNormalizeArgumentCounts(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			request := contractRequest()
			request.Args = make([]string, count)
			_, err := pluginsdk.NormalizeRequest(contractManifest(), request)
			valid := count == 1 || count == 2
			if (err == nil) != valid {
				t.Fatalf("argument count %d: %v", count, err)
			}
		})
	}
}

func TestNormalizeRejectsNonRunnableCommands(t *testing.T) {
	for _, path := range [][]string{{"example"}, {"example", "records"}} {
		if _, err := pluginsdk.NormalizeRequest(contractManifest(), pluginsdk.ExecuteRequest{CommandPath: path}); err == nil {
			t.Fatal("accepted a non-runnable command")
		}
	}
}

func TestNormalizeRequiredFlags(t *testing.T) {
	for _, flags := range []map[string]string{nil, {}, {"required": ""}} {
		request := contractRequest()
		request.Flags = flags
		if _, err := pluginsdk.NormalizeRequest(contractManifest(), request); err == nil {
			t.Fatal("accepted missing required string flag")
		}
	}
	manifest := contractManifest()
	manifest.Root.Subcommands[0].Subcommands[0].Flags[3].Default = "default-required"
	request := contractRequest()
	request.Flags = nil
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Flags["required"] != "default-required" {
		t.Fatal("required flag lost its non-empty default")
	}
	if request.Flags != nil {
		t.Fatal("normalization initialized the caller's nil map")
	}
}

func TestNormalizeFlagValidation(t *testing.T) {
	tests := []struct {
		name, flag, value string
		valid             bool
	}{
		{"bool true", "enabled", "true", true}, {"bool false", "enabled", "false", true},
		{"bool numeric", "enabled", "1", false}, {"bool uppercase", "enabled", "TRUE", false}, {"bool empty", "enabled", "", false},
		{"uint32 zero", "limit", "0", true}, {"uint32 max", "limit", "4294967295", true},
		{"uint32 overflow", "limit", "4294967296", false}, {"uint32 negative", "limit", "-1", false}, {"uint32 fraction", "limit", "1.5", false}, {"uint32 empty", "limit", "", false},
		{"uint32 huge integer", "limit", "18446744073709551616", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := contractRequest()
			request.Flags[test.flag] = test.value
			_, err := pluginsdk.NormalizeRequest(contractManifest(), request)
			if (err == nil) != test.valid {
				t.Fatalf("unexpected validation for %s=%q: %v", test.flag, test.value, err)
			}
		})
	}
}

func TestNormalizeUnknownFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		changed map[string]bool
	}{
		{"unknown value", map[string]string{"unknown": "value"}, nil},
		{"unknown empty", map[string]string{"unknown": ""}, nil},
		{"unknown changed true", nil, map[string]bool{"unknown": true}},
		{"unknown changed false", nil, map[string]bool{"unknown": false}},
		{"non-persistent changed flag", nil, map[string]bool{"root-only": true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := contractRequest()
			maps.Copy(request.Flags, test.flags)
			request.ChangedFlags = test.changed
			if _, err := pluginsdk.NormalizeRequest(contractManifest(), request); err == nil {
				t.Fatal("accepted unknown flag metadata")
			}
		})
	}
}

func TestNormalizeConfirmation(t *testing.T) {
	manifest := contractManifest()
	leaf := &manifest.Root.Subcommands[0].Subcommands[0]
	leaf.Confirm = true
	leaf.Flags = append(leaf.Flags, pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false"})
	for _, value := range []string{"", "false", "true", "TRUE", "1"} {
		t.Run(value, func(t *testing.T) {
			request := contractRequest()
			if value != "" {
				request.Flags["confirm"] = value
			}
			_, err := pluginsdk.NormalizeRequest(manifest, request)
			if (err == nil) != (value == "true") {
				t.Fatalf("incorrect confirmation validation for %q: %v", value, err)
			}
		})
	}
}

func TestNormalizeBody(t *testing.T) {
	tests := []struct {
		name  string
		body  json.RawMessage
		valid bool
	}{
		{"absent", nil, true}, {"empty non-nil", json.RawMessage{}, false},
		{"object", json.RawMessage(`{"id":9007199254740993}`), true},
		{"array", json.RawMessage(`[18446744073709551615,-9007199254740993]`), true},
		{"null", json.RawMessage(`null`), true}, {"scalar", json.RawMessage(`9007199254740993`), true},
		{"invalid", json.RawMessage(`{"id":`), false}, {"multiple values", json.RawMessage(`{} {}`), false},
		{"whitespace only", json.RawMessage(" \n"), false},
		{"exact limit", quotedBody(4 << 20), true}, {"oversized valid JSON", quotedBody((4 << 20) + 1), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := contractRequest()
			request.Body = test.body
			normalized, err := pluginsdk.NormalizeRequest(contractManifest(), request)
			if (err == nil) != test.valid {
				t.Fatalf("incorrect body validation: %v", err)
			}
			if err == nil && !bytes.Equal(normalized.Body, test.body) {
				t.Fatal("normalization changed JSON bytes or number precision")
			}
		})
	}
}

func quotedBody(size int) json.RawMessage {
	return json.RawMessage(`"` + strings.Repeat("x", size-2) + `"`)
}

func TestNormalizeRejectsDuplicateFlags(t *testing.T) {
	for _, kind := range []string{"inherited defaults", "inherited types", "inherited required", "own duplicates", "ancestor duplicates"} {
		t.Run(kind, func(t *testing.T) {
			manifest := contractManifest()
			parent := pluginsdk.FlagSpec{Name: "collision", Type: "string", Default: "parent", Persistent: true}
			child := pluginsdk.FlagSpec{Name: "collision", Type: "string", Default: "child"}
			request := contractRequest()
			switch kind {
			case "inherited types":
				parent.Type = "uint32"
				parent.Default = "10"
				request.Flags["collision"] = "42"
			case "inherited required":
				parent.Required = true
				request.Flags["collision"] = "present"
			}
			manifest.Root.Flags = append(manifest.Root.Flags, parent)
			leaf := &manifest.Root.Subcommands[0].Subcommands[0]
			switch kind {
			case "own duplicates":
				leaf.Flags = append(leaf.Flags, leaf.Flags[0])
			case "ancestor duplicates":
				manifest.Root.Subcommands[0].Flags = append(manifest.Root.Subcommands[0].Flags, parent)
			default:
				leaf.Flags = append(leaf.Flags, child)
			}
			before := jsonClone(t, request)
			if _, err := pluginsdk.NormalizeRequest(manifest, request); err == nil {
				t.Fatal("accepted duplicate flags in the effective command scope")
			}
			if !reflect.DeepEqual(request, before) {
				t.Fatal("duplicate rejection mutated caller flags")
			}
		})
	}
}

func TestNormalizeNonPersistentNameCanBeReused(t *testing.T) {
	manifest := contractManifest()
	manifest.Root.Flags = append(manifest.Root.Flags, pluginsdk.FlagSpec{Name: "local", Type: "uint32", Default: "10"})
	leaf := &manifest.Root.Subcommands[0].Subcommands[0]
	leaf.Flags = append(leaf.Flags, pluginsdk.FlagSpec{Name: "local", Type: "string", Default: "child"})
	normalized, err := pluginsdk.NormalizeRequest(manifest, contractRequest())
	if err != nil {
		t.Fatalf("ancestor non-persistent flag leaked into child scope: %v", err)
	}
	if normalized.Flags["local"] != "child" {
		t.Fatal("ancestor non-persistent default replaced the child's flag")
	}
}

func TestJSONRoundTripManifest(t *testing.T) {
	manifest := contractManifest()
	if got := jsonClone(t, manifest); !reflect.DeepEqual(got, manifest) {
		t.Fatal("manifest metadata changed during JSON round trip")
	}
}

func TestJSONRoundTripRequest(t *testing.T) {
	request := contractRequest()
	request.Body = json.RawMessage(`{"id":9007199254740993,"amount":18446744073709551615,"negative":-9007199254740993}`)
	request.Flags["enabled"] = "false"
	request.ChangedFlags = map[string]bool{"enabled": true, "limit": false}
	if got := jsonClone(t, request); !reflect.DeepEqual(got, request) {
		t.Fatal("request lost number precision, explicit false or changed flags")
	}
}

func TestJSONRoundTripResponse(t *testing.T) {
	response := pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"id":9007199254740993,"amount":18446744073709551615,"nested":[-9007199254740993]}`)}
	if got := jsonClone(t, response); !bytes.Equal(got.Data, response.Data) {
		t.Fatal("response lost JSON integer precision")
	}
	empty := pluginsdk.ExecuteResponse{}
	if got := jsonClone(t, empty); got.Data != nil {
		t.Fatal("empty response gained data")
	}
}

func jsonClone[T any](t *testing.T, value T) T {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var restored T
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	return restored
}

func TestNormalizeRejectsUnsupportedProtocolVersion(t *testing.T) {
	for _, version := range []int{-1, 0, pluginsdk.ProtocolVersion + 1} {
		t.Run(strconv.Itoa(version), func(t *testing.T) {
			manifest := contractManifest()
			manifest.ProtocolVersion = version
			request := contractRequest()
			before := jsonClone(t, request)
			if _, err := pluginsdk.NormalizeRequest(manifest, request); err == nil {
				t.Fatal("accepted an unsupported manifest protocol version")
			}
			if !reflect.DeepEqual(request, before) {
				t.Fatal("protocol rejection mutated the request")
			}
		})
	}
}

func TestNormalizeRejectsUnsupportedFlagType(t *testing.T) {
	manifest := contractManifest()
	leaf := &manifest.Root.Subcommands[0].Subcommands[0]
	leaf.Flags = append(leaf.Flags, pluginsdk.FlagSpec{Name: "unsupported", Type: "uint64", Default: "1"})
	if _, err := pluginsdk.NormalizeRequest(manifest, contractRequest()); err == nil {
		t.Fatal("accepted unsupported flag type")
	}
}

func TestNormalizeRequiredBody(t *testing.T) {
	manifest := contractManifest()
	manifest.Root.Subcommands[0].Subcommands[0].Flags[4].Required = true
	before := jsonClone(t, manifest)
	for _, test := range []struct {
		name  string
		body  json.RawMessage
		valid bool
	}{
		{"missing", nil, false},
		{"provided", json.RawMessage(`{"id":9007199254740993}`), true},
		{"invalid", json.RawMessage(`{"id":`), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := contractRequest()
			request.Body = test.body
			_, err := pluginsdk.NormalizeRequest(manifest, request)
			if (err == nil) != test.valid {
				t.Fatalf("incorrect required body validation: %v", err)
			}
			if !reflect.DeepEqual(manifest, before) {
				t.Fatal("body normalization mutated the manifest's required flag")
			}
		})
	}
}
