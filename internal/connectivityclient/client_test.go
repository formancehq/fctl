package connectivityclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestListConnectorsBuildsStackConnectivityRequest(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"items":[{"metadata":{"name":"stripe"},"spec":{"displayName":"Stripe"}}],"continue":"next"}`), nil
	})}

	got, err := New("https://stack.example/base", httpClient).ListConnectors(context.Background(), ListOptions{Limit: 25, Continue: "cursor"})

	require.NoError(t, err)
	require.Equal(t, "/base/api/connectivity/connectors", seen.URL.Path)
	require.Equal(t, "25", seen.URL.Query().Get("limit"))
	require.Equal(t, "cursor", seen.URL.Query().Get("continue"))
	require.Equal(t, "stripe", *got.Items[0].Metadata.Name)
	require.Equal(t, "next", got.Continue)
}

func TestClientMethodsRespectHTTPContracts(t *testing.T) {
	connectorName := "stripe/primary"
	instanceName := "worker/one"

	tests := []struct {
		name            string
		wantMethod      string
		wantPath        string
		wantQuery       map[string]string
		wantContentType string
		wantBody        string
		responseStatus  int
		responseBody    string
		call            func(Client) error
	}{
		{
			name:           "lists connectors",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors",
			wantQuery:      map[string]string{"connector": "stripe", "limit": "5", "continue": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"items":[]}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{Connector: "stripe", Limit: 5, Continue: "after"})
				return err
			},
		},
		{
			name:           "gets a connector using an escaped name",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/stripe%2Fprimary",
			responseStatus: http.StatusOK,
			responseBody:   `{"metadata":{"name":"stripe/primary"},"spec":{"displayName":"Stripe"}}`,
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), connectorName)
				return err
			},
		},
		{
			name:           "lists connector versions",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/stripe%2Fprimary/versions",
			responseStatus: http.StatusOK,
			responseBody:   `{"items":[{"version":"v1.0.0","image":"registry/stripe:v1.0.0"}]}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(context.Background(), connectorName)
				return err
			},
		},
		{
			name:           "gets a connector version using escaped segments",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/stripe%2Fprimary/versions/v1.0.0-rc.1",
			responseStatus: http.StatusOK,
			responseBody:   `{"version":"v1.0.0-rc.1","image":"registry/stripe:v1.0.0"}`,
			call: func(client Client) error {
				_, err := client.GetConnectorVersion(context.Background(), connectorName, "v1.0.0-rc.1")
				return err
			},
		},
		{
			name:           "lists connector instances",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectorinstances",
			wantQuery:      map[string]string{"connector": "stripe", "limit": "5", "continue": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"items":[]}`,
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(context.Background(), ListOptions{Connector: "stripe", Limit: 5, Continue: "after"})
				return err
			},
		},
		{
			name:            "creates a connector instance",
			wantMethod:      http.MethodPost,
			wantPath:        "/base/api/connectivity/connectorinstances",
			wantContentType: "application/json",
			wantBody:        `{"name":"worker","spec":{"connector":"stripe","ledger":"ledger"}}`,
			responseStatus:  http.StatusCreated,
			responseBody:    `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.CreateConnectorInstance(context.Background(), ConnectorInstanceCreate{Name: "worker", Spec: ConnectorInstanceSpec{Connector: "stripe", Ledger: "ledger"}})
				return err
			},
		},
		{
			name:           "gets a connector instance using an escaped name",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectorinstances/worker%2Fone",
			responseStatus: http.StatusOK,
			responseBody:   `{"metadata":{"name":"worker/one"},"spec":{"connector":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.GetConnectorInstance(context.Background(), instanceName)
				return err
			},
		},
		{
			name:            "patches a connector instance",
			wantMethod:      http.MethodPatch,
			wantPath:        "/base/api/connectivity/connectorinstances/worker%2Fone",
			wantContentType: "application/merge-patch+json",
			wantBody:        `{"spec":{"pollInterval":"1m"}}`,
			responseStatus:  http.StatusOK,
			responseBody:    `{"metadata":{"name":"worker/one"},"spec":{"connector":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.PatchConnectorInstance(context.Background(), instanceName, ConnectorInstancePatch{"spec": map[string]any{"pollInterval": "1m"}})
				return err
			},
		},
		{
			name:           "deletes a connector instance",
			wantMethod:     http.MethodDelete,
			wantPath:       "/base/api/connectivity/connectorinstances/worker%2Fone",
			responseStatus: http.StatusNoContent,
			call: func(client Client) error {
				return client.DeleteConnectorInstance(context.Background(), instanceName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen *http.Request
			httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				seen = req.Clone(req.Context())
				return jsonResponse(tt.responseStatus, tt.responseBody), nil
			})}

			err := tt.call(New("https://stack.example/base", httpClient))

			require.NoError(t, err)
			require.Equal(t, tt.wantMethod, seen.Method)
			require.Equal(t, tt.wantPath, seen.URL.EscapedPath())
			require.Equal(t, "application/json", seen.Header.Get("Accept"))
			for key, want := range tt.wantQuery {
				require.Equal(t, want, seen.URL.Query().Get(key), "query %q", key)
			}
			if tt.wantContentType != "" {
				require.Equal(t, tt.wantContentType, seen.Header.Get("Content-Type"))
			}
			if tt.wantBody != "" {
				body, readErr := io.ReadAll(seen.Body)
				require.NoError(t, readErr)
				require.JSONEq(t, tt.wantBody, string(body))
			}
		})
	}
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"code":"invalid","message":"connector is required","details":{"field":"connector"}}`), nil
	})}

	_, err := New("https://stack.example", httpClient).GetConnector(context.Background(), "missing")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, "invalid", apiErr.Code)
	require.Equal(t, "connector is required", apiErr.Message)
	require.Equal(t, map[string]any{"field": "connector"}, apiErr.Details)
}

// The API owns the CRD reshaping (spec.config -> spec.env/spec.files), so the
// client must put the patch on the wire in the documented API shape.
func TestPatchConnectorInstanceSendsAPIShapedConfig(t *testing.T) {
	var seenBody []byte
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		seenBody, err = io.ReadAll(req.Body)
		require.NoError(t, err)
		return jsonResponse(http.StatusOK, `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"main"}}`), nil
	})}
	password := "env-password"
	privateKey := "file-password"
	config := &ConnectorInstanceConfig{
		Env: map[string]EnvValue{
			"API_PASSWORD": {Value: &password},
		},
		Files: []FileMount{
			{Path: "/etc/plugin/key.pem", Value: &privateKey},
			{Path: "/etc/plugin/config.json", ConfigMapRef: &KeyRef{Name: "connector-config", Key: "config.json"}},
		},
	}

	_, err := New("https://stack.example", httpClient).PatchConnectorInstance(context.Background(), "worker", ConnectorInstancePatch{
		"spec": map[string]any{
			"config":       config,
			"ledger":       "main",
			"pollInterval": "15s",
		},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{
		"spec": {
			"config": {
				"env": {"API_PASSWORD": {"value": "env-password"}},
				"files": [
					{"path": "/etc/plugin/key.pem", "value": "file-password"},
					{"path": "/etc/plugin/config.json", "configMapRef": {"name": "connector-config", "key": "config.json"}}
				]
			},
			"ledger": "main",
			"pollInterval": "15s"
		}
	}`, string(seenBody))
}

func TestClientRejectsMalformedAndEmptyObjectResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		call func(Client) error
	}{
		{
			name: "malformed JSON",
			body: `{`,
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), "stripe")
				return err
			},
		},
		{
			name: "empty connector object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), "stripe")
				return err
			},
		},
		{
			name: "empty connector list object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name: "empty connector version list object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(context.Background(), "stripe")
				return err
			},
		},
		{
			name: "empty connector instance list object",
			body: "null",
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(context.Background(), ListOptions{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, tt.body), nil
			})}

			err := tt.call(New("https://stack.example", httpClient))

			require.Error(t, err)
		})
	}
}

func TestClientRejectsStructurallyInvalidSuccessResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		call   func(Client) error
	}{
		{
			name:   "connector root is not an object",
			status: http.StatusOK,
			body:   `[]`,
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "whitespace-only connector object",
			status: http.StatusOK,
			body:   "{ \n\t }",
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "connector list missing items",
			status: http.StatusOK,
			body:   `{"continue":"next"}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector instance list has null items",
			status: http.StatusOK,
			body:   `{"items":null}`,
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"displayName":"Stripe"}}`,
			call: func(client Client) error {
				_, err := client.GetConnector(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "connector list item missing metadata name",
			status: http.StatusOK,
			body:   `{"items":[{"metadata":{},"spec":{}}]}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector version missing image",
			status: http.StatusOK,
			body:   `{"version":"v1.0.0"}`,
			call: func(client Client) error {
				_, err := client.GetConnectorVersion(context.Background(), "stripe", "v1.0.0")
				return err
			},
		},
		{
			name:   "connector version list item missing version",
			status: http.StatusOK,
			body:   `{"items":[{"image":"registry/stripe:v1.0.0"}]}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "connector instance missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"connector":"stripe","ledger":"main"}}`,
			call: func(client Client) error {
				_, err := client.GetConnectorInstance(context.Background(), "worker")
				return err
			},
		},
		{
			name:   "connector instance missing connector",
			status: http.StatusCreated,
			body:   `{"metadata":{"name":"worker"},"spec":{"ledger":"main"}}`,
			call: func(client Client) error {
				_, err := client.CreateConnectorInstance(context.Background(), ConnectorInstanceCreate{Name: "worker", Spec: ConnectorInstanceSpec{Connector: "stripe", Ledger: "main"}})
				return err
			},
		},
		{
			name:   "connector instance list item missing ledger",
			status: http.StatusOK,
			body:   `{"items":[{"metadata":{"name":"worker"},"spec":{"connector":"stripe"}}]}`,
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(context.Background(), ListOptions{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(tt.status, tt.body), nil
			})}

			err := tt.call(New("https://stack.example", httpClient))

			require.Error(t, err)
		})
	}
}

func TestClientAcceptsValidEmptyLists(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"items":[]}`), nil
	})}
	client := New("https://stack.example", httpClient)

	connectors, err := client.ListConnectors(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connectors.Items)
	versions, err := client.ListConnectorVersions(context.Background(), "stripe")
	require.NoError(t, err)
	require.Empty(t, versions.Items)
	instances, err := client.ListConnectorInstances(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
}
