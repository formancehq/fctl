package instances

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func stringPtr(value string) *string {
	return &value
}

func instanceFixture(name string) connectivityclient.Instance {
	return connectivityclient.Instance{
		Metadata: connectivityclient.ObjectMeta{Name: stringPtr(name)},
		Spec: connectivityclient.InstanceSpec{
			Plugin:        "stripe",
			Version:       stringPtr("2.0.0"),
			Ledger:        "main",
			PollInterval:  stringPtr("5s"),
			StartSequence: fctl.Ptr(int64(10)),
		},
		Status: &connectivityclient.InstanceStatus{
			Phase:             stringPtr("Ready"),
			State:             stringPtr("Running"),
			ResolvedImage:     stringPtr("registry/plugin:2.0.0"),
			PluginAddress:     stringPtr("http://stripe.default.svc"),
			CurrentSequence:   fctl.Ptr(int64(42)),
			SourceTipSequence: fctl.Ptr(int64(48)),
			LastError:         stringPtr("source temporarily unavailable"),
			Message:           stringPtr("retrying ingestion"),
		},
	}
}

func instanceWithTwoFiles() *connectivityclient.Instance {
	instance := instanceFixture("stripe-eu")
	instance.Spec.Config = &connectivityclient.InstanceConfig{Files: []connectivityclient.FileMount{
		{Path: "/etc/plugin/config.yaml", Value: stringPtr("private config")},
		{Path: "/etc/plugin/ca.pem", SecretRef: &connectivityclient.KeyRef{Name: "plugin-secrets", Key: "ca.pem"}},
	}}
	return &instance
}

func factoryReturning(client connectivityclient.Client) connectivityinternal.ClientFactory {
	return func(*cobra.Command) (connectivityclient.Client, error) {
		return client, nil
	}
}

func factoryWithInstance(instance connectivityclient.Instance) connectivityinternal.ClientFactory {
	return factoryReturning(showInstanceClientMock{get: func(_ context.Context, _ string) (*connectivityclient.Instance, error) {
		return &instance, nil
	}})
}

func mockReadFile(files map[string]string) ReadFileFunc {
	return func(_ *cobra.Command, name string) (string, error) {
		value, ok := files[name]
		if !ok {
			return "", fmt.Errorf("test file %q not found", name)
		}
		return value, nil
	}
}

func mapReadFile(files map[string]string) ReadFileFunc {
	return mockReadFile(files)
}

func mockPathCompleter(paths []string) PathCompleter {
	return func(string) ([]string, error) {
		return paths, nil
	}
}

func executeCommand(command *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}

func pluginWithFileSchema() *connectivityclient.Plugin {
	return &connectivityclient.Plugin{Spec: connectivityclient.PluginSpec{
		ConfigSchema: map[string]any{
			"type": "object",
			"env": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"API_URL": map[string]any{"type": "string", "description": "API endpoint"},
				},
				"required": []any{},
			},
			"files": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"/etc/a": map[string]any{"type": "string", "description": "Primary config"},
					"/etc/b": map[string]any{"type": "string", "format": "password"},
				},
				"required": []any{"/etc/a"},
			},
		},
		Versions: []connectivityclient.VersionEntry{
			{Version: "1.0.0", Image: stringPtr("example/plugin:1.0.0")},
			{Version: "2.0.0", Image: stringPtr("example/plugin:2.0.0")},
		},
	}}
}

func pluginWithSchemaAndDefaults() *connectivityclient.Plugin {
	plugin := pluginWithFileSchema()
	plugin.Spec.ConfigSchema = map[string]any{
		"type": "object",
		"env": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"API_URL": map[string]any{"type": "string", "description": "API endpoint"},
				"TOKEN":   map[string]any{"type": "string", "format": "password", "description": "Authentication token"},
				"TIMEOUT": map[string]any{"type": "string"},
			},
			"required": []any{"API_URL", "TOKEN"},
		},
		"files": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"/etc/plugin/config.yaml": map[string]any{"type": "string", "description": "Plugin configuration"},
				"/etc/plugin/ca.pem":      map[string]any{"type": "string", "format": "password"},
			},
			"required": []any{"/etc/plugin/config.yaml"},
		},
	}
	plugin.Spec.Defaults = &connectivityclient.InstanceConfig{
		Env: map[string]connectivityclient.EnvValue{
			"API_URL": {Value: stringPtr("https://default.example")},
			"TOKEN":   {Value: stringPtr("default-token")},
			"TIMEOUT": {Value: stringPtr("30s")},
		},
		Files: []connectivityclient.FileMount{
			{Path: "/etc/plugin/config.yaml", Value: stringPtr("default: true\n")},
			{Path: "/etc/plugin/ca.pem", Value: stringPtr("default-ca")},
		},
	}
	return plugin
}
