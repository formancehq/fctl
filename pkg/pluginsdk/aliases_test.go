package pluginsdk_test

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func aliasContractManifest() pluginsdk.Manifest {
	manifest := contractManifest()
	manifest.Name = "legacy-ledger"
	manifest.Root.Use = "legacy-ledger"
	manifest.Root.Aliases = []string{"legacy_ledger"}
	group := &manifest.Root.Subcommands[0]
	group.Use, group.Aliases, group.Service = "bank-accounts", []string{"bank_accounts"}, "payments"
	group.Subcommands[0].Use = "get-metadata ID [SECOND_ID]"
	group.Subcommands[0].Aliases = []string{"get_metadata", "show"}
	return manifest
}

func TestFindAndNormalizeAliasesAtEveryLevel(t *testing.T) {
	t.Parallel()
	manifest := aliasContractManifest()
	canonical := []string{"legacy-ledger", "bank-accounts", "get-metadata"}
	for _, path := range [][]string{
		canonical,
		{"legacy_ledger", "bank_accounts", "get_metadata"},
		{"legacy-ledger", "bank_accounts", "get-metadata"},
		{"legacy_ledger", "bank-accounts", "show"},
	} {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			t.Parallel()
			testNormalizeAliasPath(t, manifest, path, canonical)
		})
	}
}

func testNormalizeAliasPath(t *testing.T, manifest pluginsdk.Manifest, path, canonical []string) {
	t.Helper()
	command, err := pluginsdk.FindCommand(manifest, path)
	if err != nil || !reflect.DeepEqual(command, manifest.Root.Subcommands[0].Subcommands[0]) {
		t.Fatalf("alias lookup lost metadata: %+v (%v)", command, err)
	}
	request := contractRequest()
	request.CommandPath = slices.Clone(path)
	before := slices.Clone(request.CommandPath)
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil || !slices.Equal(normalized.CommandPath, canonical) {
		t.Fatalf("canonical route = %v (%v)", normalized.CommandPath, err)
	}
	if !slices.Equal(request.CommandPath, before) {
		t.Fatal("normalization rewrote the caller's route")
	}
	normalized.CommandPath[0] = "modified"
	if !slices.Equal(request.CommandPath, before) {
		t.Fatal("canonical route shares caller storage")
	}
	service, err := pluginsdk.CommandService(manifest, path)
	if err != nil || service != "payments" {
		t.Fatalf("alias lookup lost the inherited service: %q (%v)", service, err)
	}
}

func TestAliasNormalizationPreservesChangedFlagsAndBody(t *testing.T) {
	t.Parallel()
	for _, changed := range []map[string]bool{nil, {}, {"enabled": true, "region": false}} {
		request := contractRequest()
		request.CommandPath = []string{"legacy_ledger", "bank_accounts", "get_metadata"}
		request.Flags["enabled"] = "false"
		request.ChangedFlags = maps.Clone(changed)
		request.Body = json.RawMessage(`{"amount":900719925474099312345678901234567890}`)
		request.Context = map[string]string{"stack": "fixture"}
		before := cloneAliasRequest(request)
		normalized, err := pluginsdk.NormalizeRequest(aliasContractManifest(), request)
		if err != nil {
			t.Fatal(err)
		}
		assertAliasNormalizationData(t, before, normalized)
		normalized.ChangedFlags = ensureChangedFlags(normalized.ChangedFlags)
		normalized.ChangedFlags["region"] = true
		if !reflect.DeepEqual(request, before) {
			t.Fatal("normalized changed flags share caller state")
		}
	}
}

func ensureChangedFlags(flags map[string]bool) map[string]bool {
	if flags == nil {
		return make(map[string]bool)
	}
	return flags
}

func cloneAliasRequest(request pluginsdk.ExecuteRequest) pluginsdk.ExecuteRequest {
	request.CommandPath = slices.Clone(request.CommandPath)
	request.Args = slices.Clone(request.Args)
	request.Flags = maps.Clone(request.Flags)
	request.ChangedFlags = maps.Clone(request.ChangedFlags)
	request.Body = bytes.Clone(request.Body)
	request.Context = maps.Clone(request.Context)
	return request
}

func assertAliasNormalizationData(t *testing.T, original, normalized pluginsdk.ExecuteRequest) {
	t.Helper()
	if !reflect.DeepEqual(normalized.ChangedFlags, original.ChangedFlags) || normalized.Flags["enabled"] != "false" || normalized.Flags["limit"] != "100" {
		t.Fatalf("default filling changed explicit flag markers: %+v", normalized)
	}
	if string(normalized.Body) != string(original.Body) || !maps.Equal(normalized.Context, original.Context) || normalized.Endpoint != original.Endpoint || !slices.Equal(normalized.Args, original.Args) {
		t.Fatal("alias normalization changed execution data")
	}
	again, err := pluginsdk.NormalizeRequest(aliasContractManifest(), normalized)
	if err != nil || !reflect.DeepEqual(again, normalized) {
		t.Fatalf("repeated normalization changed the request: %v", err)
	}
}

func TestAliasesKeepSDKGates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		edit func(*pluginsdk.Manifest, *pluginsdk.ExecuteRequest)
	}{
		{"protocol", func(m *pluginsdk.Manifest, _ *pluginsdk.ExecuteRequest) { m.ProtocolVersion++ }},
		{"arguments", func(_ *pluginsdk.Manifest, r *pluginsdk.ExecuteRequest) { r.Args = nil }},
		{"required flag", func(_ *pluginsdk.Manifest, r *pluginsdk.ExecuteRequest) { delete(r.Flags, "required") }},
		{"unknown changed flag", func(_ *pluginsdk.Manifest, r *pluginsdk.ExecuteRequest) {
			r.ChangedFlags = map[string]bool{"missing": false}
		}},
		{"invalid body", func(_ *pluginsdk.Manifest, r *pluginsdk.ExecuteRequest) { r.Body = json.RawMessage("{") }},
		{"confirmation", func(m *pluginsdk.Manifest, _ *pluginsdk.ExecuteRequest) {
			leaf := &m.Root.Subcommands[0].Subcommands[0]
			leaf.Confirm = true
			leaf.Flags = append(leaf.Flags, pluginsdk.FlagSpec{Name: "confirm", Type: "bool", Default: "false"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := aliasContractManifest()
			request := contractRequest()
			request.CommandPath = []string{"legacy_ledger", "bank_accounts", "show"}
			tc.edit(&manifest, &request)
			before := cloneAliasRequest(request)
			if _, err := pluginsdk.NormalizeRequest(manifest, request); err == nil {
				t.Fatal("alias bypassed an SDK gate")
			}
			if !reflect.DeepEqual(request, before) {
				t.Fatal("failed alias normalization mutated the caller")
			}
		})
	}
	for _, path := range [][]string{{"legacy_ledger", "bank_accounts", "missing"}, {"legacy_ledger", "bank_accounts", "show", "extra"}} {
		if _, err := pluginsdk.FindCommand(aliasContractManifest(), path); err == nil {
			t.Fatal("alias lookup accepted an unknown route")
		}
	}
}
