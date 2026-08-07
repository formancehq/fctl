package connectivityclient

import (
	"context"
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
