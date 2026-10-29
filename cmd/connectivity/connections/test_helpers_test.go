package connections

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func stringPtr(value string) *string {
	return &value
}

// connectorVersions serves the catalog half of the client interface so command
// mocks only spell out the calls their own test exercises. The zero value
// publishes two versions whose newest carries no schema, which is what a test
// that does not care about configuration wants.
type connectorVersions struct {
	connectivityclient.Client
	listVersions func(context.Context, string, connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error)
	getVersion   func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error)
}

func (v connectorVersions) ListConnectorVersions(ctx context.Context, connector string, options connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
	if v.listVersions == nil {
		return versionListFixture(), nil
	}
	return v.listVersions(ctx, connector, options)
}

func (v connectorVersions) GetConnectorVersion(ctx context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
	if v.getVersion == nil {
		return &connectivityclient.ConnectorVersion{Version: version}, nil
	}
	return v.getVersion(ctx, connector, version)
}

func versionListFixture() *connectivityclient.ConnectorVersionList {
	return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
		{Version: "1.0.0"},
		{Version: "2.0.0"},
	}}
}

func instanceFixture(name string) connectivityclient.Connection {
	return connectivityclient.Connection{Name: name,
		Configuration: connectivityclient.ConnectionConfiguration{Connector: "stripe", Version: stringPtr("2.0.0"), Ledger: stringPtr("main"), PollInterval: stringPtr("5s"), InitialCursor: []byte{0, 255, 97}},
		Status:        &connectivityclient.ConnectionStatus{Lifecycle: "active", AppliedVersion: stringPtr("2.0.0"), Ingestion: &connectivityclient.IngestionStatus{State: stringPtr("running"), Progress: &connectivityclient.IngestionProgress{Current: 42}, LastError: stringPtr("source temporarily unavailable")}},
	}
}
func instanceWithTwoFiles() *connectivityclient.Connection {
	instance := instanceFixture("stripe-eu")
	instance.Configuration.Files = map[string]string{"/etc/plugin/config.yaml": "private config", "/etc/plugin/ca.pem": "private ca"}
	return &instance
}

func factoryReturning(client connectivityclient.Client) connectivityinternal.ClientFactory {
	return func(*cobra.Command) (connectivityclient.Client, error) {
		return client, nil
	}
}

func factoryWithInstance(instance connectivityclient.Connection) connectivityinternal.ClientFactory {
	return factoryReturning(showInstanceClientMock{get: func(_ context.Context, _ string) (*connectivityclient.Connection, error) {
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

// versionWithFileSchema declares one optional environment key and two file
// keys, the first of them required.
func versionWithFileSchema() *connectivityclient.ConnectorVersion {
	return &connectivityclient.ConnectorVersion{
		Version: "2.0.0",

		ConfigSchema: map[string]any{
			"type": "object",
			"env": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"API_URL": map[string]any{"type": "string", "description": "API endpoint"},
				},
				"required": []any{},
			},
			"files": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"/etc/a": map[string]any{"type": "string", "description": "Primary config"},
					"/etc/b": map[string]any{"type": "string", "format": "password"},
				},
				"required": []any{"/etc/a"},
			},
		},
	}
}

// versionWithFullSchema is the reference schema: two required environment keys,
// one optional, one required file and one optional password file.
func versionWithFullSchema() *connectivityclient.ConnectorVersion {
	return &connectivityclient.ConnectorVersion{
		Version: "2.0.0",

		ConfigSchema: fullConfigSchema([]any{"API_URL", "TOKEN"}, []any{"/etc/plugin/config.yaml"}),
	}
}

// versionWithOptionalSchema exposes the same keys as versionWithFullSchema with
// nothing required, so input-parsing tests need not satisfy every key.
func versionWithOptionalSchema() *connectivityclient.ConnectorVersion {
	return &connectivityclient.ConnectorVersion{
		Version: "2.0.0",

		ConfigSchema: fullConfigSchema([]any{}, []any{}),
	}
}

func fullConfigSchema(requiredEnv, requiredFiles []any) map[string]any {
	return map[string]any{
		"type": "object",
		"env": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"API_URL": map[string]any{"type": "string", "description": "API endpoint"},
				"TOKEN":   map[string]any{"type": "string", "format": "password", "description": "Authentication token"},
				"TIMEOUT": map[string]any{"type": "string"},
			},
			"required": requiredEnv,
		},
		"files": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"/etc/plugin/config.yaml": map[string]any{"type": "string", "description": "Connector configuration"},
				"/etc/plugin/ca.pem":      map[string]any{"type": "string", "format": "password"},
			},
			"required": requiredFiles,
		},
	}
}

func setInstanceConfig(instance *connectivityclient.Connection, config *connectivityclient.ConnectionConfig) {
	if config == nil {
		instance.Configuration.Env, instance.Configuration.Files = nil, nil
		return
	}
	instance.Configuration.Env, instance.Configuration.Files = config.Env, config.Files
}
