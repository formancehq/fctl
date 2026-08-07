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

func TestListPluginsBuildsStackConnectivityRequest(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"items":[{"metadata":{"name":"stripe"},"spec":{"image":"img"}}],"continue":"next"}`), nil
	})}

	got, err := New("https://stack.example/base", httpClient).ListPlugins(context.Background(), ListOptions{Limit: 25, Continue: "cursor"})

	require.NoError(t, err)
	require.Equal(t, "/base/api/connectivity/plugins", seen.URL.Path)
	require.Equal(t, "25", seen.URL.Query().Get("limit"))
	require.Equal(t, "cursor", seen.URL.Query().Get("continue"))
	require.Equal(t, "stripe", *got.Items[0].Metadata.Name)
	require.Equal(t, "next", got.Continue)
}

func TestClientMethodsRespectHTTPContracts(t *testing.T) {
	pluginName := "stripe/primary"
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
			name:           "lists plugins",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/plugins",
			wantQuery:      map[string]string{"plugin": "stripe", "limit": "5", "continue": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"items":[]}`,
			call: func(client Client) error {
				_, err := client.ListPlugins(context.Background(), ListOptions{Plugin: "stripe", Limit: 5, Continue: "after"})
				return err
			},
		},
		{
			name:           "gets a plugin using an escaped name",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/plugins/stripe%2Fprimary",
			responseStatus: http.StatusOK,
			responseBody:   `{"metadata":{"name":"stripe/primary"},"spec":{"image":"image"}}`,
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), pluginName)
				return err
			},
		},
		{
			name:           "lists instances",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/instances",
			wantQuery:      map[string]string{"plugin": "stripe", "limit": "5", "continue": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"items":[]}`,
			call: func(client Client) error {
				_, err := client.ListInstances(context.Background(), ListOptions{Plugin: "stripe", Limit: 5, Continue: "after"})
				return err
			},
		},
		{
			name:            "creates an instance",
			wantMethod:      http.MethodPost,
			wantPath:        "/base/api/connectivity/instances",
			wantContentType: "application/json",
			wantBody:        `{"name":"worker","spec":{"plugin":"stripe","ledger":"ledger"}}`,
			responseStatus:  http.StatusCreated,
			responseBody:    `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.CreateInstance(context.Background(), InstanceCreate{Name: "worker", Spec: InstanceSpec{Plugin: "stripe", Ledger: "ledger"}})
				return err
			},
		},
		{
			name:           "gets an instance using an escaped name",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/instances/worker%2Fone",
			responseStatus: http.StatusOK,
			responseBody:   `{"metadata":{"name":"worker/one"},"spec":{"plugin":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.GetInstance(context.Background(), instanceName)
				return err
			},
		},
		{
			name:            "patches an instance",
			wantMethod:      http.MethodPatch,
			wantPath:        "/base/api/connectivity/instances/worker%2Fone",
			wantContentType: "application/merge-patch+json",
			wantBody:        `{"spec":{"pollInterval":"1m"}}`,
			responseStatus:  http.StatusOK,
			responseBody:    `{"metadata":{"name":"worker/one"},"spec":{"plugin":"stripe","ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.PatchInstance(context.Background(), instanceName, InstancePatch{"spec": map[string]any{"pollInterval": "1m"}})
				return err
			},
		},
		{
			name:           "deletes an instance",
			wantMethod:     http.MethodDelete,
			wantPath:       "/base/api/connectivity/instances/worker%2Fone",
			responseStatus: http.StatusNoContent,
			call: func(client Client) error {
				return client.DeleteInstance(context.Background(), instanceName)
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
		return jsonResponse(http.StatusBadRequest, `{"code":"invalid","message":"plugin is required","details":{"field":"plugin"}}`), nil
	})}

	_, err := New("https://stack.example", httpClient).GetPlugin(context.Background(), "missing")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, "invalid", apiErr.Code)
	require.Equal(t, "plugin is required", apiErr.Message)
	require.Equal(t, map[string]any{"field": "plugin"}, apiErr.Details)
}

func TestPatchInstanceAdaptsAPIConfigToPinnedCRDWireShape(t *testing.T) {
	var seenBody []byte
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		seenBody, err = io.ReadAll(req.Body)
		require.NoError(t, err)
		return jsonResponse(http.StatusOK, `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"main"}}`), nil
	})}
	password := "env-password"
	privateKey := "file-password"
	config := &InstanceConfig{
		Env: map[string]EnvValue{
			"API_PASSWORD": {Value: &password},
		},
		Files: []FileMount{
			{Path: "/etc/plugin/key.pem", Value: &privateKey},
			{Path: "/etc/plugin/config.json", ConfigMapRef: &KeyRef{Name: "plugin-config", Key: "config.json"}},
		},
	}

	_, err := New("https://stack.example", httpClient).PatchInstance(context.Background(), "worker", InstancePatch{
		"spec": map[string]any{
			"config":       config,
			"ledger":       "main",
			"pollInterval": "15s",
		},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{
		"spec": {
			"env": {"API_PASSWORD": {"value": "env-password"}},
			"files": [
				{"path": "/etc/plugin/key.pem", "value": "file-password"},
				{"path": "/etc/plugin/config.json", "configMapRef": {"name": "plugin-config", "key": "config.json"}}
			],
			"ledger": "main",
			"pollInterval": "15s"
		}
	}`, string(seenBody))
	require.NotContains(t, string(seenBody), `"config"`)
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
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name: "empty plugin object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name: "empty plugin list object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.ListPlugins(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name: "empty instance list object",
			body: "null",
			call: func(client Client) error {
				_, err := client.ListInstances(context.Background(), ListOptions{})
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
			name:   "plugin root is not an object",
			status: http.StatusOK,
			body:   `[]`,
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "whitespace-only plugin object",
			status: http.StatusOK,
			body:   "{ \n\t }",
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "plugin list missing items",
			status: http.StatusOK,
			body:   `{"continue":"next"}`,
			call: func(client Client) error {
				_, err := client.ListPlugins(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "instance list has null items",
			status: http.StatusOK,
			body:   `{"items":null}`,
			call: func(client Client) error {
				_, err := client.ListInstances(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "plugin missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"image":"image"}}`,
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "plugin missing image",
			status: http.StatusOK,
			body:   `{"metadata":{"name":"stripe"},"spec":{}}`,
			call: func(client Client) error {
				_, err := client.GetPlugin(context.Background(), "stripe")
				return err
			},
		},
		{
			name:   "plugin list item missing required fields",
			status: http.StatusOK,
			body:   `{"items":[{"metadata":{"name":"stripe"},"spec":{}}]}`,
			call: func(client Client) error {
				_, err := client.ListPlugins(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "instance missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"plugin":"stripe","ledger":"main"}}`,
			call: func(client Client) error {
				_, err := client.GetInstance(context.Background(), "worker")
				return err
			},
		},
		{
			name:   "instance missing plugin",
			status: http.StatusCreated,
			body:   `{"metadata":{"name":"worker"},"spec":{"ledger":"main"}}`,
			call: func(client Client) error {
				_, err := client.CreateInstance(context.Background(), InstanceCreate{Name: "worker", Spec: InstanceSpec{Plugin: "stripe", Ledger: "main"}})
				return err
			},
		},
		{
			name:   "instance list item missing ledger",
			status: http.StatusOK,
			body:   `{"items":[{"metadata":{"name":"worker"},"spec":{"plugin":"stripe"}}]}`,
			call: func(client Client) error {
				_, err := client.ListInstances(context.Background(), ListOptions{})
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

	plugins, err := client.ListPlugins(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, plugins.Items)
	instances, err := client.ListInstances(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
}
