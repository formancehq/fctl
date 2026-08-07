package instances

import (
	"fmt"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func stringPtr(value string) *string {
	return &value
}

func mapReadFile(files map[string]string) ReadFileFunc {
	return func(_ *cobra.Command, name string) (string, error) {
		value, ok := files[name]
		if !ok {
			return "", fmt.Errorf("test file %q not found", name)
		}
		return value, nil
	}
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
