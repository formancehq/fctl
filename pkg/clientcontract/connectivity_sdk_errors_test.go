package clientcontract

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/retry"
)

type connectivityErrorTransport func(*http.Request) (*http.Response, error)

func (f connectivityErrorTransport) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestConnectivityHTTPFailuresReturnSDKError(t *testing.T) {
	connector, ledger, pollInterval := "example", "main", "30s"
	operationsUnderTest := []struct {
		name, method, path, contractPath string
		call                             func(context.Context, *formance.FormanceV1) (any, error)
	}{
		{
			name: "ConnectivityInfo", method: http.MethodGet, path: "/_info",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.ConnectivityInfo(ctx)
			},
		},
		{
			name: "CreateConnection", method: http.MethodPost, path: "/connections",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.CreateConnection(ctx, connectivity.ConnectionCreate{
					Name: "example-main",
					ConnectionConfiguration: connectivity.ConnectionCreateConnectionConfiguration{
						Connector: connector, Ledger: ledger, Channel: connectivity.ConnectionCreateChannelStable.ToPointer(),
					},
				}, nil)
			},
		},
		{
			name: "DeleteConnection", method: http.MethodDelete, path: "/connections/example-main", contractPath: "/connections/{name}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.DeleteConnection(ctx, operations.DeleteConnectionRequest{Name: "example-main"}, nil)
			},
		},
		{
			name: "GetConnection", method: http.MethodGet, path: "/connections/example-main", contractPath: "/connections/{name}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.GetConnection(ctx, operations.GetConnectionRequest{Name: "example-main"}, nil)
			},
		},
		{
			name: "GetConnector", method: http.MethodGet, path: "/connectors/example", contractPath: "/connectors/{name}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.GetConnector(ctx, operations.GetConnectorRequest{Name: connector}, nil)
			},
		},
		{
			name: "GetConnectorFacets", method: http.MethodGet, path: "/connectors/_facets",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.GetConnectorFacets(ctx, operations.GetConnectorFacetsRequest{}, nil)
			},
		},
		{
			name: "GetConnectorVersion", method: http.MethodGet, path: "/connectors/example/versions/v1.0.0", contractPath: "/connectors/{name}/versions/{version}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.GetConnectorVersion(ctx, operations.GetConnectorVersionRequest{Name: connector, Version: "v1.0.0"}, nil)
			},
		},
		{
			name: "GetQueryCapabilities", method: http.MethodGet, path: "/_query/capabilities",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.GetQueryCapabilities(ctx, nil)
			},
		},
		{
			name: "Healthcheck", method: http.MethodGet, path: "/_healthcheck",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.Healthcheck(ctx)
			},
		},
		{
			name: "ListConnections", method: http.MethodGet, path: "/connections",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.ListConnections(ctx, operations.ListConnectionsRequest{}, nil)
			},
		},
		{
			name: "ListConnectorVersions", method: http.MethodGet, path: "/connectors/example/versions", contractPath: "/connectors/{name}/versions",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.ListConnectorVersions(ctx, operations.ListConnectorVersionsRequest{Name: connector}, nil)
			},
		},
		{
			name: "ListConnectors", method: http.MethodGet, path: "/connectors",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.ListConnectors(ctx, operations.ListConnectorsRequest{}, nil)
			},
		},
		{
			name: "PatchConnection", method: http.MethodPatch, path: "/connections/example-main", contractPath: "/connections/{name}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.PatchConnection(ctx, operations.PatchConnectionRequest{
					Name: "example-main", ConnectionPatch: connectivity.ConnectionPatch{
						Configuration: &connectivity.Configuration{PollInterval: &pollInterval},
					},
				}, nil)
			},
		},
		{
			name: "ReplaceConnection", method: http.MethodPut, path: "/connections/example-main", contractPath: "/connections/{name}",
			call: func(ctx context.Context, client *formance.FormanceV1) (any, error) {
				return client.ReplaceConnection(ctx, operations.ReplaceConnectionRequest{
					Name: "example-main", ConnectionUpdate: connectivity.ConnectionUpdate{
						ConnectionConfiguration: connectivity.ConnectionConfiguration{
							Connector: &connector, Ledger: &ledger, Channel: connectivity.ChannelStable.ToPointer(),
						},
					},
				}, nil)
			},
		},
	}
	contractJSON, err := os.ReadFile("../../openapi/stack.json")
	require.NoError(t, err)
	var contract struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(contractJSON, &contract))
	contractOperations := map[string]bool{}
	for path, item := range contract.Paths {
		if !strings.HasPrefix(path, "/api/connectivity/") {
			continue
		}
		for method := range item {
			switch method {
			case "get", "post", "put", "patch", "delete", "head", "options", "trace":
				contractOperations[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	testedOperations := map[string]bool{}
	for _, operation := range operationsUnderTest {
		path := operation.contractPath
		if path == "" {
			path = operation.path
		}
		testedOperations[operation.method+" /api/connectivity"+path] = true
	}
	require.Equal(t, contractOperations, testedOperations, "every Connectivity operation in the source contract must be covered")

	bodies := []struct {
		name, contentType, body string
	}{
		{
			name: "JSON error", contentType: "application/json",
			body: `{"errorCode":"example_failure","errorMessage":"request failed","details":"{\"field\":\"configuration\"}","unknown":9007199254740993}`,
		},
		{name: "invalid JSON", contentType: "application/json", body: `{"errorCode":`},
		{name: "plain text", contentType: "text/plain", body: "upstream unavailable\n"},
		{name: "empty JSON", contentType: "application/json"},
		{name: "empty body without content type"},
	}

	// The generation overlay selects one error channel for every HTTP failure,
	// including malformed JSON and response bodies outside the server's contract.
	for _, operation := range operationsUnderTest {
		t.Run(operation.name, func(t *testing.T) {
			for _, body := range bodies {
				t.Run(body.name, func(t *testing.T) {
					for status := http.StatusBadRequest; status <= 599; status++ {
						calls := 0
						var rawResponse *http.Response
						client := formance.New(
							formance.WithServerURL("https://stack.example.test/proxy"),
							formance.WithRetryConfig(retry.Config{Strategy: "none"}),
							formance.WithClient(connectivityErrorTransport(func(request *http.Request) (*http.Response, error) {
								calls++
								require.Equal(t, operation.method, request.Method)
								require.Equal(t, "/proxy/api/connectivity"+operation.path, request.URL.Path)
								headers := http.Header{}
								if body.contentType != "" {
									headers.Set("Content-Type", body.contentType)
								}
								rawResponse = &http.Response{
									StatusCode: status, Header: headers, Request: request,
									Body: io.NopCloser(strings.NewReader(body.body)),
								}
								return rawResponse, nil
							})),
						)
						response, err := operation.call(t.Context(), client.Connectivity.V1)
						require.Error(t, err, "HTTP %d", status)
						require.Nil(t, response, "HTTP %d", status)
						var sdkError *sdkerrors.SDKError
						require.ErrorAs(t, err, &sdkError, "HTTP %d", status)
						require.Equal(t, status, sdkError.StatusCode)
						require.Equal(t, body.body, sdkError.Body)
						require.Same(t, rawResponse, sdkError.RawResponse)
						raw, readErr := io.ReadAll(sdkError.RawResponse.Body)
						require.NoError(t, readErr)
						require.Equal(t, body.body, string(raw))
						require.Equal(t, 1, calls, "HTTP %d must not be retried", status)
					}
				})
			}
		})
	}
}
