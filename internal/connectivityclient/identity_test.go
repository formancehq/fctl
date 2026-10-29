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

// These are consumer receipts for request presence and normalized server replies;
// identity enforcement itself belongs to the service at the pinned revision.
func TestPATCHIdentityPresenceAndTypedRejections(t *testing.T) {
	for _, field := range []string{"connector", "ledger"} {
		for _, tc := range []struct {
			name    string
			present bool
			value   any
			status  int
		}{
			{"omitted", false, nil, 200},
			{"first assignment", true, "initial", 200},
			{"same assignment", true, "initial", 200},
			{"different assignment", true, "different", 422},
			{"clear assigned", true, nil, 422},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				response := instanceResponse
				if tc.status == 422 {
					response = `{"code":"` + field + `_immutable","message":"cannot change identity","details":{"field":"` + field + `"}}`
				}
				server, requests := contractServer(t, tc.status, response)
				configuration := map[string]any{"suspend": false, "env": map[string]any{"EMPTY": "", "OLD": nil}}
				if tc.present {
					configuration[field] = tc.value
				}
				patch := ConnectionPatch{"configuration": configuration}
				result, err := New(server.URL, server.Client()).PatchConnection(t.Context(), "worker", patch)
				if tc.status == 422 {
					var apiErr *APIError
					require.ErrorAs(t, err, &apiErr)
					require.Equal(t, 422, apiErr.StatusCode)
					require.Equal(t, field+"_immutable", apiErr.Code)
					require.Equal(t, field, apiErr.Details["field"])
					require.Nil(t, result)
				} else {
					require.NoError(t, err)
				}
				request := <-requests
				expected, err := json.Marshal(patch)
				require.NoError(t, err)
				require.JSONEq(t, string(expected), string(request.body))
			})
		}
	}
}
