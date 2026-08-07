package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestShowPluginRendersMetadataVersionsCapabilitiesStatusAndSchemaWithoutDefaults(t *testing.T) {
	created := time.Date(2026, time.August, 7, 10, 30, 0, 0, time.UTC)
	plugin := pluginFixture("stripe")
	plugin.Metadata.Namespace = fctl.Ptr("formance")
	plugin.Metadata.ResourceVersion = fctl.Ptr("42")
	plugin.Metadata.UID = fctl.Ptr("plugin-uid")
	plugin.Metadata.CreationTimestamp = &created
	plugin.Metadata.Labels = map[string]string{"region": "eu"}
	plugin.Metadata.Annotations = map[string]string{"owner": "platform"}
	plugin.Spec.Version = fctl.Ptr("2.0.0")
	plugin.Spec.DocsURL = fctl.Ptr("https://docs.example/stripe")
	plugin.Spec.Versions = []connectivityclient.VersionEntry{
		{Version: "1.0.0", Digest: fctl.Ptr("sha256:one")},
		{Version: "2.0.0", Image: fctl.Ptr("registry/plugin:2")},
	}
	plugin.Spec.ConfigSchema = map[string]any{
		"type": "object",
		"env": map[string]any{
			"type":       "object",
			"required":   []any{"API_KEY"},
			"properties": map[string]any{"API_KEY": map[string]any{"type": "string", "format": "password", "description": "API credential"}},
		},
		"files": map[string]any{
			"type":       "object",
			"properties": map[string]any{"/etc/plugin.yaml": map[string]any{"type": "string", "description": "Plugin settings"}},
		},
	}
	plugin.Spec.Defaults = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: fctl.Ptr("must-not-leak")},
	}}
	plugin.Status.Message = fctl.Ptr("Catalog entry is healthy")

	client := pluginClientMock{get: func(_ context.Context, name string) (*connectivityclient.Plugin, error) {
		if name != "stripe" {
			t.Fatalf("GetPlugin name = %q, want stripe", name)
		}
		return &plugin, nil
	}}
	command := NewShowCommand(factoryReturning(client))
	output, err := executeCommand(command, "stripe")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}

	for _, expected := range []string{
		"stripe", "formance", "42", "plugin-uid", created.Format(time.RFC3339), "region=eu", "owner=platform",
		"registry/plugin", "Plugin description", "https://docs.example/stripe", "2.0.0", "1.0.0", "sha256:one",
		"payments, webhooks", "Ready", "Catalog entry is healthy",
		"Configuration Schema", "API_KEY", "environment", "required", "password", "API credential",
		"/etc/plugin.yaml", "file", "Plugin settings",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
	if strings.Contains(output, "must-not-leak") {
		t.Fatalf("plain output leaked an inline default value:\n%s", output)
	}
}

func TestShowPluginJSONPreservesCompleteModel(t *testing.T) {
	plugin := pluginFixture("stripe")
	plugin.Spec.Defaults = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: fctl.Ptr("json-keeps-full-model")},
	}}
	client := pluginClientMock{get: func(_ context.Context, _ string) (*connectivityclient.Plugin, error) {
		return &plugin, nil
	}}

	command := NewShowCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "--output", "json", "stripe")
	if err != nil {
		t.Fatalf("execute JSON show command: %v", err)
	}

	var envelope struct {
		Data ShowStore `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	if !reflect.DeepEqual(envelope.Data.Plugin, plugin) {
		t.Fatalf("JSON plugin = %#v, want complete model %#v", envelope.Data.Plugin, plugin)
	}
}

func TestShowPluginReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityclient.Client, error)
		want    string
	}{
		"factory": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
		"API": {
			factory: factoryReturning(pluginClientMock{get: func(context.Context, string) (*connectivityclient.Plugin, error) {
				return nil, errors.New("plugin unavailable")
			}}),
			want: "plugin unavailable",
		},
		"empty response": {
			factory: factoryReturning(pluginClientMock{get: func(context.Context, string) (*connectivityclient.Plugin, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewShowCommand(test.factory), "stripe")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}

func TestShowPluginSummarizesLegacyFlatSchema(t *testing.T) {
	plugin := pluginFixture("legacy")
	plugin.Status = nil
	plugin.Spec.ConfigSchema = map[string]any{
		"type":     "object",
		"required": []string{"ENDPOINT"},
		"properties": map[string]any{
			"ENDPOINT": map[string]any{"type": "string", "description": "Service endpoint"},
		},
	}
	client := pluginClientMock{get: func(context.Context, string) (*connectivityclient.Plugin, error) {
		return &plugin, nil
	}}

	output, err := executeCommand(NewShowCommand(factoryReturning(client)), "legacy")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}
	for _, expected := range []string{"ENDPOINT", "environment", "required", "string", "Service endpoint"} {
		if !strings.Contains(output, expected) {
			t.Errorf("legacy schema output missing %q:\n%s", expected, output)
		}
	}
}
