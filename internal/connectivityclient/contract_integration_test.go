package connectivityclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// These portable HTTP fixtures follow Connectivity's public contract at
// OpenAPI SHA-256 f5468cd103ff1800effc7c431778a6d66b2f6c1fe054591e861efa8e6119148d.
// Source and official SDK: Connectivity 1d0e16394f80863b1cb451819c76f5d37d3ccded.
// They exercise the SDK-backed facade, not a running Connectivity implementation.
type wireRequest struct {
	method, path, contentType string
	body                      []byte
	err                       error
}

func contractServer(t *testing.T, status int, response string) (*httptest.Server, <-chan wireRequest) {
	t.Helper()
	requests := make(chan wireRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		requests <- wireRequest{req.Method, req.URL.EscapedPath(), req.Header.Get("Content-Type"), body, err}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server, requests
}

func TestGivenPublicContract_WhenInstanceCreated_ThenLiteralConfigurationReachesHTTP(t *testing.T) {
	for _, selector := range []string{"version", "channel"} {
		t.Run(selector, func(t *testing.T) {
			// Given a public response that omits confidential values and optional status.
			response := `{"name":"worker","configuration":{"connector":"stripe","ledger":"main","` + selector + `":"` + map[string]string{"version": "v1.0.0", "channel": "stable"}[selector] + `","env":{"EMPTY":""}}}`
			server, requests := contractServer(t, http.StatusCreated, response)
			configuration := ConnectionConfiguration{
				Connector: "stripe", Ledger: "main", InitialCursor: []byte{0, 0, 0, 0, 0, 0, 3, 232}, PollInterval: pointer("30s"), Suspend: pointer(false),
				Env: map[string]string{"API_PASSWORD": "synthetic-password", "EMPTY": ""}, Files: map[string]string{"/etc/plugin/key": "synthetic-key"},
			}
			if selector == "version" {
				configuration.Version = pointer("v1.0.0")
			} else {
				configuration.Channel = pointer("stable")
			}
			// When the typed request crosses the real HTTP transport.
			created, err := New(server.URL, server.Client()).CreateConnection(t.Context(), ConnectionCreate{Name: "worker", Configuration: configuration})
			require.NoError(t, err)
			request := <-requests
			require.NoError(t, request.err)
			require.Equal(t, http.MethodPost, request.method)
			require.Equal(t, "/api/connectivity/connections", request.path)
			require.Equal(t, "application/json", request.contentType)
			selectorJSON := `"version":"v1.0.0"`
			if selector == "channel" {
				selectorJSON = `"channel":"stable"`
			}
			// Then the wire has no metadata, reference objects or value wrappers.
			require.JSONEq(t, `{"name":"worker","configuration":{"connector":"stripe","ledger":"main",`+selectorJSON+`,"initialCursor":"AAAAAAAAA+g=","pollInterval":"30s","suspend":false,"env":{"API_PASSWORD":"synthetic-password","EMPTY":""},"files":{"/etc/plugin/key":"synthetic-key"}}}`, string(request.body))
			require.Equal(t, "worker", created.Name)
			require.Equal(t, map[string]string{"EMPTY": ""}, created.Configuration.Env)
			require.NotContains(t, created.Configuration.Env, "API_PASSWORD")
			require.Nil(t, created.Status)
		})
	}
}

func TestGivenPublicContract_WhenSelectorAndMapsPatched_ThenNullsAndLiteralsReachHTTP(t *testing.T) {
	// Given an existing instance whose confidential values are absent on reads.
	server, requests := contractServer(t, 200, `{"name":"worker","configuration":{"connector":"stripe","ledger":"main","channel":"stable","env":{"EMPTY":""},"files":{"/etc/plugin/key":"replacement"}}}`)
	// When switching from a pinned version without replacing the full config.
	patched, err := New(server.URL, server.Client()).PatchConnection(t.Context(), "worker", ConnectionPatch{
		"configuration": map[string]any{
			"version": nil, "channel": "stable", "suspend": false, "initialCursor": []byte{0, 0, 0, 0, 0, 0, 3, 232},
			"env": map[string]any{"OLD": nil, "EMPTY": ""}, "files": map[string]any{"/etc/plugin/old": nil, "/etc/plugin/key": "replacement"},
		},
	})
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.Equal(t, http.MethodPatch, request.method)
	require.Equal(t, "/api/connectivity/connections/worker", request.path)
	require.Equal(t, "application/merge-patch+json", request.contentType)
	// Then null deletion markers survive independently of omitted secret values.
	require.JSONEq(t, `{"configuration":{"version":null,"channel":"stable","suspend":false,"initialCursor":"AAAAAAAAA+g=","env":{"OLD":null,"EMPTY":""},"files":{"/etc/plugin/old":null,"/etc/plugin/key":"replacement"}}}`, string(request.body))
	require.Nil(t, patched.Configuration.Version)
	require.Equal(t, "stable", *patched.Configuration.Channel)
	require.Equal(t, "replacement", patched.Configuration.Files["/etc/plugin/key"])
}

func TestGivenPublicContract_WhenOptionalConfigurationCleared_ThenNullsReachHTTP(t *testing.T) {
	server, requests := contractServer(t, 200, instanceResponse)
	_, err := New(server.URL, server.Client()).PatchConnection(t.Context(), "worker", ConnectionPatch{
		"configuration": map[string]any{"initialCursor": nil, "pollInterval": nil, "env": nil, "files": nil},
	})
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.JSONEq(t, `{"configuration":{"initialCursor":null,"pollInterval":null,"env":null,"files":null}}`, string(request.body))
}

func TestGivenPublicContract_WhenConfigurationReplaced_ThenPUTUsesUpdateEnvelope(t *testing.T) {
	server, requests := contractServer(t, 200, instanceResponse)
	_, err := New(server.URL, server.Client()).ReplaceConnection(t.Context(), "worker", ConnectionUpdate{
		Configuration: ConnectionConfiguration{Connector: "stripe", Ledger: "main", Version: pointer("v1.0.0"), Env: map[string]string{"TOKEN": "complete-secret"}},
	})
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.Equal(t, http.MethodPut, request.method)
	require.Equal(t, "/api/connectivity/connections/worker", request.path)
	require.Equal(t, "application/json", request.contentType)
	require.JSONEq(t, `{"configuration":{"connector":"stripe","ledger":"main","version":"v1.0.0","env":{"TOKEN":"complete-secret"}}}`, string(request.body))
}

func TestGivenPublicContract_WhenMapsAreEmpty_ThenPatchPreservesEmptyObjects(t *testing.T) {
	server, requests := contractServer(t, 200, instanceResponse)
	_, err := New(server.URL, server.Client()).PatchConnection(t.Context(), "worker", ConnectionPatch{
		"configuration": map[string]any{"env": map[string]string{}, "files": map[string]string{}},
	})
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.JSONEq(t, `{"configuration":{"env":{},"files":{}}}`, string(request.body))
}
