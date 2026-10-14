package connectivityclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestGivenTenantContract_WhenInstanceIsCreated_ThenSpecRoundTrips(t *testing.T) {
	for _, selector := range []string{"version", "channel"} {
		t.Run(selector, func(t *testing.T) {
			// Given the tenant contract pinned to server revision 4084c566.
			value, empty, interval := "inline", "", "30s"
			mode, suspend := int32(384), false
			spec := ConnectorInstanceSpec{
				ConnectorRef: &ObjectRef{Name: "stripe"}, ConnectivityRef: &ObjectRef{Name: "stack"},
				Ledger: "main", InitialCursor: []byte{0, 0, 0, 0, 0, 0, 3, 232},
				PollInterval: &interval, Suspend: &suspend,
				Env: map[string]EnvValue{
					"INLINE": {Value: &value}, "EMPTY": {Value: &empty},
					"SECRET": {SecretRef: &KeyRef{Name: "credentials", Key: "api-key"}},
					"CONFIG": {ConfigMapRef: &KeyRef{Name: "settings", Key: "region"}},
				},
				Files: []FileMount{
					{Path: "/etc/plugin/inline", Value: &value, Mode: &mode},
					{Path: "/etc/plugin/secret", SecretRef: &KeyRef{Name: "credentials", Key: "key.pem"}},
					{Path: "/etc/plugin/config", ConfigMapRef: &KeyRef{Name: "settings", Key: "config.json"}},
				},
			}
			selected, wantSelector := "v1.0.0", `"version":"v1.0.0"`
			if selector == "channel" {
				selected, wantSelector = "stable", `"channel":"stable"`
				spec.Channel = &selected
			} else {
				spec.Version = &selected
			}
			wantSpec := `{"connectorRef":{"name":"stripe"},"connectivityRef":{"name":"stack"},` + wantSelector + `,
				"ledger":"main","initialCursor":"AAAAAAAAA+g=","pollInterval":"30s","suspend":false,
				"env":{"INLINE":{"value":"inline"},"EMPTY":{"value":""},"SECRET":{"secretRef":{"name":"credentials","key":"api-key"}},"CONFIG":{"configMapRef":{"name":"settings","key":"region"}}},
				"files":[{"path":"/etc/plugin/inline","value":"inline","mode":384},{"path":"/etc/plugin/secret","secretRef":{"name":"credentials","key":"key.pem"}},{"path":"/etc/plugin/config","configMapRef":{"name":"settings","key":"config.json"}}]}`
			response := `{"metadata":{"name":"worker","namespace":"tenant","resourceVersion":"7","uid":"instance-id","creationTimestamp":"2026-10-14T10:00:00Z","labels":{"team":"payments"},"annotations":{"owner":"operations"}},"spec":` + wantSpec + `,"status":{"phase":"Running","resolvedConnectorRef":"stripe","resolvedVersion":"v1.0.0","currentSequence":1000,"sourceTipSequence":2000}}`
			server, requests := contractServer(t, http.StatusCreated, response)

			// When a typed create request crosses the HTTP transport.
			created, err := New(server.URL, server.Client()).CreateConnectorInstance(t.Context(), ConnectorInstanceCreate{
				Name: "worker", Labels: map[string]string{"team": "payments"}, Annotations: map[string]string{"owner": "operations"}, Spec: spec,
			})

			// Then only tenant fields are sent, and binary cursors and projections decode unchanged.
			require.NoError(t, err)
			request := <-requests
			require.NoError(t, request.err)
			require.Equal(t, http.MethodPost, request.method)
			require.Equal(t, "/api/connectivity/connectorinstances", request.path)
			require.Equal(t, "application/json", request.contentType)
			require.JSONEq(t, `{"name":"worker","labels":{"team":"payments"},"annotations":{"owner":"operations"},"spec":`+wantSpec+`}`, string(request.body))
			require.Equal(t, spec, created.Spec)
			require.Equal(t, spec.Configuration(), created.Spec.Configuration())
			require.Equal(t, "tenant", *created.Metadata.Namespace)
			require.Equal(t, "7", *created.Metadata.ResourceVersion)
			require.Equal(t, "instance-id", *created.Metadata.UID)
			require.NotNil(t, created.Metadata.CreationTimestamp)
			require.Equal(t, map[string]string{"team": "payments"}, created.Metadata.Labels)
			require.Equal(t, map[string]string{"owner": "operations"}, created.Metadata.Annotations)
			require.Equal(t, "stripe", *created.Status.ResolvedConnectorRef)
			require.Equal(t, int64(1000), *created.Status.CurrentSequence)
			require.Equal(t, int64(2000), *created.Status.SourceTipSequence)
		})
	}
}

func TestGivenTenantContract_WhenSpecIsMergePatched_ThenNullsAndEmptyArraysReachWire(t *testing.T) {
	// Given an existing bound instance and a response with the requested selector.
	response := `{"metadata":{"name":"worker"},"spec":{"connectorRef":{"name":"stripe"},"connectivityRef":{"name":"stack"},"ledger":"main","channel":"stable","initialCursor":"AAAAAAAAA+g=","suspend":false,"env":{"REGION":{"configMapRef":{"name":"settings","key":"region"}}},"files":[]}}`
	server, requests := contractServer(t, http.StatusOK, response)

	// When a patch removes a version and an env entry, replaces files with an
	// empty array, and sends an opaque cursor without a configuration wrapper.
	patched, err := New(server.URL, server.Client()).PatchConnectorInstance(t.Context(), "worker", ConnectorInstancePatch{
		"spec": map[string]any{
			"connectorRef": &ObjectRef{Name: "stripe"}, "connectivityRef": &ObjectRef{Name: "stack"},
			"ledger": "main", "version": nil, "channel": "stable", "suspend": false,
			"initialCursor": []byte{0, 0, 0, 0, 0, 0, 3, 232},
			"env":           map[string]any{"OLD": nil, "REGION": EnvValue{ConfigMapRef: &KeyRef{Name: "settings", Key: "region"}}},
			"files":         []FileMount{},
		},
	})

	// Then merge-patch deletion markers survive and the response uses the same model.
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.Equal(t, http.MethodPatch, request.method)
	require.Equal(t, "/api/connectivity/connectorinstances/worker", request.path)
	require.Equal(t, "application/merge-patch+json", request.contentType)
	require.JSONEq(t, `{"spec":{"connectorRef":{"name":"stripe"},"connectivityRef":{"name":"stack"},"ledger":"main","version":null,"channel":"stable","suspend":false,"initialCursor":"AAAAAAAAA+g=","env":{"OLD":null,"REGION":{"configMapRef":{"name":"settings","key":"region"}}},"files":[]}}`, string(request.body))
	require.Nil(t, patched.Spec.Version)
	require.Equal(t, "stable", *patched.Spec.Channel)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 3, 232}, patched.Spec.InitialCursor)
	require.Equal(t, "stack", patched.Spec.ConnectivityRef.Name)
	require.NotNil(t, patched.Spec.Files)
	require.Empty(t, patched.Spec.Files)
	require.False(t, *patched.Spec.Suspend)
}

func TestGivenTenantContract_WhenConfigurationIsCleared_ThenTopLevelNullsReachWire(t *testing.T) {
	// Given an instance whose optional configuration can be removed.
	server, requests := contractServer(t, http.StatusOK, `{"metadata":{"name":"worker"},"spec":{"connectorRef":{"name":"stripe"},"channel":"stable","ledger":"main"}}`)

	// When the caller explicitly removes optional tenant fields.
	patched, err := New(server.URL, server.Client()).PatchConnectorInstance(t.Context(), "worker", ConnectorInstancePatch{
		"spec": map[string]any{"initialCursor": nil, "pollInterval": nil, "env": nil, "files": nil},
	})

	// Then null differs from omission on the wire, and absent fields decode safely.
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.JSONEq(t, `{"spec":{"initialCursor":null,"pollInterval":null,"env":null,"files":null}}`, string(request.body))
	require.Nil(t, patched.Spec.InitialCursor)
	require.Nil(t, patched.Spec.PollInterval)
	require.Equal(t, &ConnectorInstanceConfig{}, patched.Spec.Configuration())
}
