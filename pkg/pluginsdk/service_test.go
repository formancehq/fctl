package pluginsdk_test

import (
	"bytes"
	"maps"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func serviceManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "multi", Version: "test", Service: "fallback", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{
		Use: "multi", Service: "root-service", Runnable: true,
		Subcommands: []pluginsdk.CommandSpec{
			{Use: "default", Runnable: true},
			{Use: "branch", Service: "branch-service", Subcommands: []pluginsdk.CommandSpec{
				{Use: "inherited", Runnable: true},
				{Use: "leaf", Service: "leaf-service", Runnable: true},
			}},
		},
	}}
}

func TestCommandServiceInheritance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ path, want string }{
		{"multi", "root-service"}, {"multi default", "root-service"},
		{"multi branch", "branch-service"}, {"multi branch inherited", "branch-service"},
		{"multi branch leaf", "leaf-service"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			got, err := pluginsdk.CommandService(serviceManifest(), strings.Fields(tc.path))
			if err != nil || got != tc.want {
				t.Fatalf("service = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
	manifest := serviceManifest()
	manifest.Root.Service = ""
	got, err := pluginsdk.CommandService(manifest, []string{"multi", "default"})
	if err != nil || got != manifest.Service {
		t.Fatalf("manifest fallback = %q, %v", got, err)
	}
}

func TestCommandServiceRejectsUnknownPaths(t *testing.T) {
	t.Parallel()
	for _, path := range [][]string{nil, {}, {"wrong"}, {"multi", "missing"}, {"multi", "branch", "missing"}, {"multi", "branch", "leaf", "extra"}} {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			t.Parallel()
			got, err := pluginsdk.CommandService(serviceManifest(), path)
			if err == nil || got != "" {
				t.Fatalf("invalid path %v returned %q, %v", path, got, err)
			}
		})
	}
}

func TestRequireTrueValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, kind, value, fallback string
		ok                          bool
	}{
		{"enabled", "bool", "true", "false", true},
		{"false", "bool", "false", "false", false},
		{"omitted", "bool", "", "false", false},
		{"default-enabled", "bool", "", "true", true},
		{"malformed", "bool", "yes", "false", false},
		{"string-true", "string", "true", "", false},
		{"integer", "uint32", "1", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := serviceManifest()
			manifest.Root.Flags = []pluginsdk.FlagSpec{{Name: "experimental", Type: tc.kind, Default: tc.fallback, Persistent: true, RequireTrue: true}}
			request := pluginsdk.ExecuteRequest{CommandPath: []string{"multi", "branch", "leaf"}, Flags: map[string]string{}}
			if tc.value != "" {
				request.Flags["experimental"] = tc.value
			}
			got, err := pluginsdk.NormalizeRequest(manifest, request)
			if (err == nil) != tc.ok {
				t.Fatalf("NormalizeRequest error = %v; want success %v", err, tc.ok)
			}
			if tc.ok && got.Flags["experimental"] != "true" {
				t.Fatal("enabled gate was lost")
			}
			if !tc.ok && !strings.Contains(err.Error(), "experimental") {
				t.Fatalf("unactionable error: %v", err)
			}
		})
	}
}

func TestNormalizationPreservesCallerMapsAndBody(t *testing.T) {
	t.Parallel()
	manifest := serviceManifest()
	manifest.Root.Flags = []pluginsdk.FlagSpec{
		{Name: "experimental", Type: "bool", Default: "false", RequireTrue: true},
		{Name: "data", Type: "string", Body: true, Required: true},
		{Name: "limit", Type: "uint32", Default: "10"},
	}
	request := pluginsdk.ExecuteRequest{CommandPath: []string{"multi"}, Flags: map[string]string{"experimental": "true"}, ChangedFlags: map[string]bool{"experimental": true}, Body: []byte(`{"amount":90071992547409930001}`), Context: map[string]string{"stack": "fixture"}}
	flags, metadata, body := maps.Clone(request.Flags), maps.Clone(request.Context), bytes.Clone(request.Body)
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Flags["limit"] != "10" || !bytes.Equal(normalized.Body, body) {
		t.Fatal("defaults or precise body lost")
	}
	normalized.Flags["experimental"] = "false"
	normalized.Context["stack"] = "mutated"
	if !maps.Equal(request.Flags, flags) || !maps.Equal(request.Context, metadata) || !bytes.Equal(request.Body, body) {
		t.Fatal("normalization or handler mutation changed the caller request")
	}
}
