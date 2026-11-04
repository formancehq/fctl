package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func stringPtr(value string) *string {
	return &value
}

// connectorVersions serves the catalog half of the client interface so command
// mocks only spell out the calls their own test exercises. The zero value
// publishes two versions whose newest carries no schema, which is what a test
// that does not care about configuration wants.
type connectorVersions struct {
	connectivityinternal.API
	listVersions func(context.Context, string, connectivityinternal.ListOptions) (*testConnectorVersionList, error)
	getVersion   func(context.Context, string, string) (*connectivity.ConnectorVersion, error)
}

func (v connectorVersions) ListConnectorVersions(ctx context.Context, request operations.ListConnectorVersionsRequest, security *operations.ListConnectorVersionsSecurity, opts ...operations.Option) (*operations.ListConnectorVersionsResponse, error) {
	var data *testConnectorVersionList
	var err error
	if v.listVersions == nil {
		data = versionListFixture()
	} else {
		data, err = v.listVersions(ctx, request.Name, listOptions(request.PageSize, request.Cursor, request.Query))
	}
	response := &operations.ListConnectorVersionsResponse{StatusCode: 200}
	if data != nil {
		response.ConnectorVersionCursor = &connectivity.ConnectorVersionCursor{Cursor: connectivity.ConnectorVersionCursorCursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}
	}
	return response, err
}

func (v connectorVersions) GetConnectorVersion(ctx context.Context, request operations.GetConnectorVersionRequest, security *operations.GetConnectorVersionSecurity, opts ...operations.Option) (*operations.GetConnectorVersionResponse, error) {
	if v.getVersion == nil {
		return &operations.GetConnectorVersionResponse{StatusCode: 200, ConnectorVersion: &connectivity.ConnectorVersion{Version: request.Version}}, nil
	}
	data, err := v.getVersion(ctx, request.Name, request.Version)
	return &operations.GetConnectorVersionResponse{StatusCode: 200, ConnectorVersion: data}, err
}

func versionListFixture() *testConnectorVersionList {
	return &testConnectorVersionList{Items: []connectivity.ConnectorVersionSummary{
		{Version: "1.0.0"},
		{Version: "2.0.0"},
	}}
}

func instanceFixture(name string) connectivity.Connection {
	return connectivity.Connection{Name: name,
		ConnectionConfiguration: connectivity.ConnectionConfiguration{Connector: stringPtr("stripe"), Version: stringPtr("2.0.0"), Ledger: stringPtr("main"), PollInterval: stringPtr("5s"), InitialCursor: stringPtr("AP9h")},
		ConnectionStatus:        &connectivity.ConnectionStatus{Lifecycle: "active", AppliedVersion: stringPtr("2.0.0"), IngestionStatus: &connectivity.IngestionStatus{State: ptr(connectivity.State("running")), IngestionProgress: &connectivity.IngestionProgress{Current: 42}, LastError: stringPtr("source temporarily unavailable")}},
	}
}
func instanceWithTwoFiles() *connectivity.Connection {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.Files = map[string]string{"/etc/plugin/config.yaml": "private config", "/etc/plugin/ca.pem": "private ca"}
	return &instance
}

func factoryReturning(client connectivityinternal.API) connectivityinternal.ClientFactory {
	return func(*cobra.Command) (connectivityinternal.API, error) {
		return client, nil
	}
}

func factoryWithInstance(instance connectivity.Connection) connectivityinternal.ClientFactory {
	return factoryReturning(showInstanceClientMock{get: func(_ context.Context, _ string) (*connectivity.Connection, error) {
		return &instance, nil
	}})
}

func mockReadFile(files map[string]string) configuration.ReadFileFunc {
	return func(_ *cobra.Command, name string) (string, error) {
		value, ok := files[name]
		if !ok {
			return "", fmt.Errorf("test file %q not found", name)
		}
		return value, nil
	}
}

func mapReadFile(files map[string]string) configuration.ReadFileFunc {
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
func versionWithFileSchema() *connectivity.ConnectorVersion {
	return &connectivity.ConnectorVersion{
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
func versionWithFullSchema() *connectivity.ConnectorVersion {
	return &connectivity.ConnectorVersion{
		Version: "2.0.0",

		ConfigSchema: fullConfigSchema([]any{"API_URL", "TOKEN"}, []any{"/etc/plugin/config.yaml"}),
	}
}

// versionWithOptionalSchema exposes the same keys as versionWithFullSchema with
// nothing required, so input-parsing tests need not satisfy every key.
func versionWithOptionalSchema() *connectivity.ConnectorVersion {
	return &connectivity.ConnectorVersion{
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

func setInstanceConfig(instance *connectivity.Connection, config *connectivity.ConnectionConfiguration) {
	if config == nil {
		instance.ConnectionConfiguration.Env, instance.ConnectionConfiguration.Files = nil, nil
		return
	}
	instance.ConnectionConfiguration.Env, instance.ConnectionConfiguration.Files = config.Env, config.Files
}

// These fixture pages keep command tests focused on pagination behavior; the
// mocks above return the generated SDK response envelopes.
type testConnectionList struct {
	Items    []connectivity.Connection
	PageSize int32
	HasMore  bool
	Next     string
}
type testConnectorList struct {
	Items    []connectivity.Connector
	PageSize int32
	HasMore  bool
	Next     string
}
type testConnectorVersionList struct {
	Items    []connectivity.ConnectorVersionSummary
	PageSize int32
	HasMore  bool
	Next     string
}

func listOptions(size *int, cursor, query *string) connectivityinternal.ListOptions {
	options := connectivityinternal.ListOptions{Cursor: stringValue(cursor), Query: stringValue(query)}
	if size != nil {
		options.PageSize = int32(*size)
	}
	return options
}
func createdConfiguration(in connectivity.ConnectionCreateConnectionConfiguration) connectivity.ConnectionConfiguration {
	out := connectivity.ConnectionConfiguration{Connector: &in.Connector, Ledger: &in.Ledger, Env: in.Env, Files: in.Files, Version: in.Version, InitialCursor: in.InitialCursor, PollInterval: in.PollInterval, Suspend: in.Suspend}
	if in.Channel != nil {
		out.Channel = ptr(connectivity.Channel(*in.Channel))
	}
	return out
}

type connectionRoundTripper func(*http.Request) (*http.Response, error)

func (f connectionRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
func sdkPatchBody(ctx context.Context, request operations.PatchConnectionRequest) (map[string]any, error) {
	var body map[string]any
	api := connectivityinternal.NewAPI("https://stack.example.test", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"name":"fixture","configuration":{}}`)), Request: r}, nil
	})})
	_, err := api.PatchConnection(ctx, request, nil)
	return body, err
}
