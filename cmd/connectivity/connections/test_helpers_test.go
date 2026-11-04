package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func stringPtr(value string) *string {
	return &value
}

// connectorVersions serves catalog HTTP responses so command fixtures only
// spell out the calls their own test exercises. The zero value
// publishes two versions whose newest carries no schema, which is what a test
// that does not care about configuration wants.
type connectorVersions struct {
	listVersions func(context.Context, string, connectivityinternal.ListOptions) (*testConnectorVersionList, error)
	getVersion   func(context.Context, string, string) (*connectivity.ConnectorVersion, error)
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

func factoryWithInstance(instance connectivity.Connection) testSDKFactory {
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

type testSDKFactory func(*cobra.Command) (*sdk.FormanceV1, error)

func transportSDK(uri string, client *http.Client) *sdk.FormanceV1 {
	return connectivityinternal.PatchAPI(&fctl.StackClients{URI: uri, HTTPClient: client})
}
func mockSDK(transport http.RoundTripper) *sdk.FormanceV1 {
	return transportSDK("https://stack.example.test", &http.Client{Transport: transport})
}
func factoryReturning(transport http.RoundTripper) testSDKFactory {
	return factoryWithSDK(mockSDK(transport))
}
func factoryWithSDK(client *sdk.FormanceV1) testSDKFactory {
	return func(*cobra.Command) (*sdk.FormanceV1, error) { return client, nil }
}
func requestOptions(r *http.Request) connectivityinternal.ListOptions {
	size, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 32)
	return connectivityinternal.ListOptions{PageSize: int32(size), Cursor: r.URL.Query().Get("cursor"), Query: r.URL.Query().Get("query")}
}
func mockHTTPResponse(r *http.Request, status int, data any, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonHTTPResponse(r, status, string(body)), nil
}
func mockConnectionPage(r *http.Request, data *testConnectionList, err error) (*http.Response, error) {
	if data == nil {
		return mockHTTPResponse(r, 200, nil, err)
	}
	return mockHTTPResponse(r, 200, connectivity.ConnectionCursor{Cursor: connectivity.Cursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}, err)
}
func (v connectorVersions) RoundTrip(r *http.Request) (*http.Response, error) {
	prefix := "/api/connectivity/connectors/"
	suffix := strings.TrimPrefix(r.URL.EscapedPath(), prefix)
	parts := strings.SplitN(suffix, "/versions", 2)
	if len(parts) != 2 || r.Method != http.MethodGet {
		return nil, fmt.Errorf("unexpected request %s %s", r.Method, r.URL)
	}
	name, _ := url.PathUnescape(parts[0])
	if parts[1] == "" {
		data := versionListFixture()
		var err error
		if v.listVersions != nil {
			data, err = v.listVersions(r.Context(), name, requestOptions(r))
		}
		if data == nil {
			return mockHTTPResponse(r, 200, nil, err)
		}
		return mockHTTPResponse(r, 200, connectivity.ConnectorVersionCursor{Cursor: connectivity.ConnectorVersionCursorCursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}, err)
	}
	version, _ := url.PathUnescape(strings.TrimPrefix(parts[1], "/"))
	data := &connectivity.ConnectorVersion{Version: version}
	var err error
	if v.getVersion != nil {
		data, err = v.getVersion(r.Context(), name, version)
	}
	return mockHTTPResponse(r, 200, data, err)
}
func requestConnectionName(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/api/connectivity/connections/")
}
func (m listInstanceClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	data, err := m.list(r.Context(), requestOptions(r))
	return mockConnectionPage(r, data, err)
}
func (m showInstanceClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	data, err := m.get(r.Context(), requestConnectionName(r))
	return mockHTTPResponse(r, 200, data, err)
}
func (m completionClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path == "/api/connectivity/connections" {
		data, err := m.listInstances(r.Context(), requestOptions(r))
		return mockConnectionPage(r, data, err)
	}
	return m.connectorVersions.RoundTrip(r)
}
func (m configureClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "/connectors/") {
		return m.connectorVersions.RoundTrip(r)
	}
	if r.URL.Path == "/api/connectivity/connections" {
		data, err := m.listInstances(r.Context(), requestOptions(r))
		return mockConnectionPage(r, data, err)
	}
	if r.Method == http.MethodGet {
		data, err := m.getInstance(r.Context(), requestConnectionName(r))
		return mockHTTPResponse(r, 200, data, err)
	}
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		return nil, err
	}
	data, err := m.patch(r.Context(), requestConnectionName(r), patch)
	return mockHTTPResponse(r, 200, data, err)
}
func (m suspensionClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodGet {
		data, err := m.listInstances(r.Context(), requestOptions(r))
		return mockConnectionPage(r, data, err)
	}
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		return nil, err
	}
	data, err := m.patch(r.Context(), requestConnectionName(r), patch)
	return mockHTTPResponse(r, 200, data, err)
}
func (m uninstallClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodGet {
		data, err := m.listInstances(r.Context(), requestOptions(r))
		return mockConnectionPage(r, data, err)
	}
	return mockHTTPResponse(r, 204, nil, m.delete(r.Context(), requestConnectionName(r)))
}
func (m mutationClientMock) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "/versions") {
		return m.connectorVersions.RoundTrip(r)
	}
	if strings.HasSuffix(r.URL.Path, "/_query/capabilities") {
		data, err := m.capabilities(r.Context())
		return mockHTTPResponse(r, 200, data, err)
	}
	if r.Method == http.MethodGet {
		data, err := m.listConnectors(r.Context(), requestOptions(r))
		if data == nil {
			return mockHTTPResponse(r, 200, nil, err)
		}
		return mockHTTPResponse(r, 200, connectivity.ConnectorCursor{Cursor: connectivity.ConnectorCursorCursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}, err)
	}
	var body connectivity.ConnectionCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	data, err := m.create(r.Context(), body)
	return mockHTTPResponse(r, 201, data, err)
}
