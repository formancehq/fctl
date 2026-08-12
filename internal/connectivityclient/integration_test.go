package connectivityclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGivenConnectivityServer_WhenLifecycleMethodsRun_ThenContractIsRespected(t *testing.T) {
	// Given a connectivity server with the public lifecycle routes.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectors":
			require.Equal(t, "application/json", req.Header.Get("Accept"))
			_, _ = io.WriteString(w, `{"items":[{"metadata":{"name":"stripe"},"spec":{"displayName":"Stripe"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectors/stripe":
			_, _ = io.WriteString(w, `{"metadata":{"name":"stripe"},"spec":{"displayName":"Stripe"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectors/stripe/versions":
			_, _ = io.WriteString(w, `{"items":[{"version":"v1.0.0","image":"registry/stripe:v1.0.0"}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectors/stripe/versions/v1.0.0":
			_, _ = io.WriteString(w, `{"version":"v1.0.0","image":"registry/stripe:v1.0.0","configSchema":{"env":{"properties":{}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectorinstances":
			_, _ = io.WriteString(w, `{"items":[]}`)
		case req.Method == http.MethodPost && req.URL.Path == "/api/connectivity/connectorinstances":
			require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/connectorinstances/worker":
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/api/connectivity/connectorinstances/worker":
			require.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodDelete && req.URL.Path == "/api/connectivity/connectorinstances/worker":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	// When lifecycle methods use the server.
	connectors, err := client.ListConnectors(ctx, ListOptions{})
	require.NoError(t, err)
	connector, err := client.GetConnector(ctx, "stripe")
	require.NoError(t, err)
	versions, err := client.ListConnectorVersions(ctx, "stripe")
	require.NoError(t, err)
	version, err := client.GetConnectorVersion(ctx, "stripe", "v1.0.0")
	require.NoError(t, err)
	instances, err := client.ListConnectorInstances(ctx, ListOptions{})
	require.NoError(t, err)
	created, err := client.CreateConnectorInstance(ctx, ConnectorInstanceCreate{Name: "worker", Spec: ConnectorInstanceSpec{Connector: "stripe", Ledger: "ledger"}})
	require.NoError(t, err)
	fetched, err := client.GetConnectorInstance(ctx, "worker")
	require.NoError(t, err)
	patched, err := client.PatchConnectorInstance(ctx, "worker", ConnectorInstancePatch{"spec": map[string]any{"pollInterval": "1m"}})
	require.NoError(t, err)
	err = client.DeleteConnectorInstance(ctx, "worker")
	require.NoError(t, err)

	// Then each response is decoded from the API contract.
	require.Len(t, connectors.Items, 1)
	require.Equal(t, "stripe", *connectors.Items[0].Metadata.Name)
	require.Equal(t, "stripe", *connector.Metadata.Name)
	require.Len(t, versions.Items, 1)
	require.Equal(t, "v1.0.0", versions.Items[0].Version)
	require.Equal(t, "v1.0.0", version.Version)
	require.NotEmpty(t, version.ConfigSchema)
	require.Empty(t, instances.Items)
	require.Equal(t, "worker", *created.Metadata.Name)
	require.Equal(t, "worker", *fetched.Metadata.Name)
	require.Equal(t, "worker", *patched.Metadata.Name)
}

func TestGivenConnectivityAPI_WhenConfigIsPatched_ThenAPIShapeIsSent(t *testing.T) {
	// Given a server asserting the documented ConnectorInstancePatch shape.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodPatch, req.Method)
		require.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))

		var patch map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&patch))
		spec, ok := patch["spec"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, spec, "env")
		require.NotContains(t, spec, "files")
		require.Equal(t, "30s", spec["pollInterval"])
		config, ok := spec["config"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, map[string]any{
			"API_PASSWORD": map[string]any{"value": "env-password"},
		}, config["env"])
		require.Equal(t, []any{
			map[string]any{"path": "/etc/plugin/key.pem", "value": "file-password"},
			map[string]any{"path": "/etc/plugin/config.json", "value": "complete-file-array"},
		}, config["files"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"connector":"stripe","ledger":"main"}}`)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	envPassword := "env-password"
	filePassword := "file-password"
	otherFile := "complete-file-array"

	// When an API-shaped config patch contains password-compatible inline values.
	_, err := client.PatchConnectorInstance(context.Background(), "worker", ConnectorInstancePatch{
		"spec": map[string]any{
			"config": &ConnectorInstanceConfig{
				Env: map[string]EnvValue{
					"API_PASSWORD": {Value: &envPassword},
				},
				Files: []FileMount{
					{Path: "/etc/plugin/key.pem", Value: &filePassword},
					{Path: "/etc/plugin/config.json", Value: &otherFile},
				},
			},
			"pollInterval": "30s",
		},
	})

	// Then the API receives spec.config verbatim and preserves the full files array.
	require.NoError(t, err)
}
