package connectivityclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportedConnectionReadsOmitUnassignedIdentities(t *testing.T) {
	for _, configuration := range []string{`{}`, `{"connector":"stripe"}`, `{"ledger":"main"}`} {
		t.Run(configuration, func(t *testing.T) {
			body := `{"name":"worker","configuration":` + configuration + `}`
			c := New("https://stack.example", responseClient(200, body))
			connection, err := c.GetConnection(t.Context(), "worker")
			require.NoError(t, err)
			encoded, err := json.Marshal(connection)
			require.NoError(t, err)
			require.JSONEq(t, body, string(encoded))
			page, err := New("https://stack.example", responseClient(200, `{"cursor":{"data":[`+body+`]}}`)).ListConnections(t.Context(), ListOptions{})
			require.NoError(t, err)
			require.Len(t, page.Items, 1)
		})
	}
}

func TestPUTPreservesOmittedIdentitiesOnWire(t *testing.T) {
	for _, config := range []ConnectionConfiguration{
		{Version: pointer("v1.0.0")},
		{Connector: "stripe", Ledger: pointer("main"), Version: pointer("v1.0.0")},
	} {
		server, requests := contractServer(t, 200, instanceResponse)
		_, err := New(server.URL, server.Client()).ReplaceConnection(t.Context(), "worker", ConnectionUpdate{Configuration: config})
		require.NoError(t, err)
		request := <-requests
		require.Equal(t, http.MethodPut, request.method)
		expected, err := json.Marshal(ConnectionUpdate{Configuration: config})
		require.NoError(t, err)
		require.JSONEq(t, string(expected), string(request.body))
		if config.Ledger == nil {
			require.JSONEq(t, `{"configuration":{"version":"v1.0.0"}}`, string(request.body))
		}
	}
}

func TestPATCHRejectsAnyBindingFieldBeforeSending(t *testing.T) {
	for _, field := range []string{"connector", "ledger"} {
		for _, tc := range []struct {
			name  string
			value any
		}{
			{"same or initial value", "initial"},
			{"different value", "different"},
			{"empty", ""},
			{"null", nil},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				calls := 0
				httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return jsonResponse(200, instanceResponse), nil
				})}
				result, err := New("https://stack.example", httpClient).PatchConnection(t.Context(), "worker", ConnectionPatch{
					"configuration": map[string]any{field: tc.value, "suspend": false},
				})
				require.Nil(t, result)
				require.ErrorContains(t, err, "invalid connectivity patch configuration")
				require.ErrorContains(t, err, `unknown field "`+field+`"`)
				require.Zero(t, calls, "forbidden fields must be rejected, not silently discarded by SDK conversion")
			})
		}
	}
}

func TestPATCHNormalizesServerUnsupportedFieldError(t *testing.T) {
	server, requests := contractServer(t, 400, `{"code":"unsupported_field","message":"Unsupported tenant field","details":{"field":"configuration"}}`)
	result, err := New(server.URL, server.Client()).PatchConnection(t.Context(), "worker", ConnectionPatch{
		"configuration": map[string]any{"suspend": false},
	})
	require.Nil(t, result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 400, apiErr.StatusCode)
	require.Equal(t, "unsupported_field", apiErr.Code)
	require.Equal(t, "Unsupported tenant field", apiErr.Message)
	require.Equal(t, map[string]any{"field": "configuration"}, apiErr.Details)
	request := <-requests
	require.Equal(t, http.MethodPatch, request.method)
	require.JSONEq(t, `{"configuration":{"suspend":false}}`, string(request.body))
}
