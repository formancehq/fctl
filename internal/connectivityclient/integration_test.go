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
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/plugins":
			require.Equal(t, "application/json", req.Header.Get("Accept"))
			_, _ = io.WriteString(w, `{"items":[{"metadata":{"name":"stripe"},"spec":{"image":"image"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/plugins/stripe":
			_, _ = io.WriteString(w, `{"metadata":{"name":"stripe"},"spec":{"image":"image"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/instances":
			_, _ = io.WriteString(w, `{"items":[]}`)
		case req.Method == http.MethodPost && req.URL.Path == "/api/connectivity/instances":
			require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/connectivity/instances/worker":
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/api/connectivity/instances/worker":
			require.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))
			_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"ledger"}}`)
		case req.Method == http.MethodDelete && req.URL.Path == "/api/connectivity/instances/worker":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	// When lifecycle methods use the server.
	plugins, err := client.ListPlugins(ctx, ListOptions{})
	require.NoError(t, err)
	plugin, err := client.GetPlugin(ctx, "stripe")
	require.NoError(t, err)
	instances, err := client.ListInstances(ctx, ListOptions{})
	require.NoError(t, err)
	created, err := client.CreateInstance(ctx, InstanceCreate{Name: "worker", Spec: InstanceSpec{Plugin: "stripe", Ledger: "ledger"}})
	require.NoError(t, err)
	fetched, err := client.GetInstance(ctx, "worker")
	require.NoError(t, err)
	patched, err := client.PatchInstance(ctx, "worker", InstancePatch{"spec": map[string]any{"pollInterval": "1m"}})
	require.NoError(t, err)
	err = client.DeleteInstance(ctx, "worker")
	require.NoError(t, err)

	// Then each response is decoded from the API contract.
	require.Len(t, plugins.Items, 1)
	require.Equal(t, "stripe", *plugins.Items[0].Metadata.Name)
	require.Equal(t, "stripe", *plugin.Metadata.Name)
	require.Empty(t, instances.Items)
	require.Equal(t, "worker", *created.Metadata.Name)
	require.Equal(t, "worker", *fetched.Metadata.Name)
	require.Equal(t, "worker", *patched.Metadata.Name)
}

func TestGivenPinnedConnectivityPatchHandler_WhenAPIConfigIsPatched_ThenCRDShapeIsSent(t *testing.T) {
	// Given a server that applies the PATCH body directly to the pinned Instance CRD.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodPatch, req.Method)
		require.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))

		var patch map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&patch))
		spec, ok := patch["spec"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, spec, "config")
		require.Equal(t, "30s", spec["pollInterval"])
		require.Equal(t, map[string]any{
			"API_PASSWORD": map[string]any{"value": "env-password"},
		}, spec["env"])
		require.Equal(t, []any{
			map[string]any{"path": "/etc/plugin/key.pem", "value": "file-password"},
			map[string]any{"path": "/etc/plugin/config.json", "value": "complete-file-array"},
		}, spec["files"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"metadata":{"name":"worker"},"spec":{"plugin":"stripe","ledger":"main"}}`)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	envPassword := "env-password"
	filePassword := "file-password"
	otherFile := "complete-file-array"

	// When an API-shaped config patch contains password-compatible inline values.
	_, err := client.PatchInstance(context.Background(), "worker", InstancePatch{
		"spec": map[string]any{
			"config": &InstanceConfig{
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

	// Then the client sends root CRD env/files fields for materialisation and preserves the full files array.
	require.NoError(t, err)
}
