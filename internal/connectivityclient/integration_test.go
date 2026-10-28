package connectivityclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func responseClient(status int, body string) *http.Client {
	return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(status, body), nil })}
}

func TestGivenPublicContract_WhenStatusRead_ThenRecordedProgressAndErrorsDecode(t *testing.T) {
	server, requests := contractServer(t, 200, `{"name":"worker","configuration":{"ledger":"main"},"status":{"appliedVersion":"v1.2.0","lifecycle":"failed","ingestion":{"state":"error","progress":{"current":9007199254740993},"lastError":"historical error"},"error":{"code":"service_error","message":"Service operation failed"},"suspendedBy":[{"source":"policy","name":"maintenance"}]}}`)
	instance, err := New(server.URL, server.Client()).GetConnectorInstance(t.Context(), "worker")
	require.NoError(t, err)
	request := <-requests
	require.NoError(t, request.err)
	require.Equal(t, http.MethodGet, request.method)
	require.Equal(t, "/api/connectivity/connector-instances/worker", request.path)
	require.Equal(t, int64(9007199254740993), instance.Status.Ingestion.Progress.Current)
	require.Equal(t, "error", *instance.Status.Ingestion.State)
	require.Equal(t, "failed", instance.Status.Lifecycle)
	require.Equal(t, "service_error", instance.Status.Error.Code)
	require.Equal(t, "policy", instance.Status.SuspendedBy[0].Source)
}

func TestGivenPublicContract_WhenSuspensionPatched_ThenExplicitBooleanReachesHTTP(t *testing.T) {
	for _, suspend := range []bool{false, true} {
		name := "resume"
		literal := "false"
		if suspend {
			name = "suspend"
			literal = "true"
		}
		t.Run(name, func(t *testing.T) {
			server, requests := contractServer(t, 200, instanceResponse)
			_, err := New(server.URL, server.Client()).PatchConnectorInstance(t.Context(), "worker", ConnectorInstancePatch{"configuration": map[string]any{"suspend": suspend}})
			require.NoError(t, err)
			request := <-requests
			require.NoError(t, request.err)
			require.JSONEq(t, `{"configuration":{"suspend":`+literal+`}}`, string(request.body))
		})
	}
}

func TestGivenPublicContract_WhenRequestFails_ThenHTTPStatusAndDiagnosticSurvive(t *testing.T) {
	for _, tt := range []struct {
		name, body, code string
		status           int
	}{
		{"unauthorized", "", "", 401},
		{"unknown field", `{"code":"unsupported_field","message":"Unsupported tenant field","details":{"field":"metadata"}}`, "unsupported_field", 400},
		{"immutable ledger", `{"code":"ledger_immutable","message":"Ledger cannot change"}`, "ledger_immutable", 422},
		{"unavailable", `{"code":"unavailable","message":"Service unavailable"}`, "unavailable", 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server, requests := contractServer(t, tt.status, tt.body)
			result, err := New(server.URL, server.Client()).GetConnectorInstance(t.Context(), "worker")
			require.Nil(t, result)
			require.NoError(t, (<-requests).err)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tt.status, apiErr.StatusCode)
			require.Equal(t, tt.code, apiErr.Code)
			require.NotEmpty(t, apiErr.Message)
		})
	}
}

func TestWriteValidationPreventsInvalidConfigurationRequests(t *testing.T) {
	valid := ConnectorInstanceConfiguration{Connector: "stripe", Ledger: "main", Version: pointer("v1.0.0")}
	cases := map[string]ConnectorInstanceConfiguration{}
	config := valid
	config.Connector = ""
	cases["missing connector"] = config
	config = valid
	config.Ledger = ""
	cases["missing ledger"] = config
	config = valid
	config.Version = nil
	cases["missing selector"] = config
	config = valid
	config.Channel = pointer("stable")
	cases["two selectors"] = config
	config = valid
	config.Version = pointer("")
	cases["empty version"] = config
	config = valid
	config.Version = nil
	config.Channel = pointer("latest")
	cases["invalid channel"] = config
	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				t.Error("invalid request reached transport")
				return jsonResponse(500, ""), nil
			})})
			created, err := c.CreateConnectorInstance(t.Context(), ConnectorInstanceCreate{Name: "worker", Configuration: config})
			require.Error(t, err)
			require.Nil(t, created)
			replaced, err := c.ReplaceConnectorInstance(t.Context(), "worker", ConnectorInstanceUpdate{Configuration: config})
			require.Error(t, err)
			require.Nil(t, replaced)
		})
	}
	_, err := New("https://stack.example", nil).CreateConnectorInstance(t.Context(), ConnectorInstanceCreate{Configuration: valid})
	require.ErrorContains(t, err, "name is required")
}

func TestPatchRejectsAdministrativeAndNonliteralInputsBeforeHTTP(t *testing.T) {
	for _, patch := range []ConnectorInstancePatch{
		nil, {"spec": map[string]any{"suspend": true}}, {"metadata": map[string]any{"name": "worker"}},
		{"configuration": nil}, {"configuration": []string{}},
		{"configuration": map[string]any{"connectorRef": map[string]any{"name": "stripe"}}},
		{"configuration": map[string]any{"env": map[string]any{"TOKEN": map[string]any{"secretRef": map[string]any{"name": "secret", "key": "token"}}}}},
		{"configuration": map[string]any{"files": []any{map[string]any{"path": "/etc/key", "value": "contents"}}}},
		{"configuration": map[string]any{"channel": "latest"}},
		{"configuration": map[string]any{"initialCursor": "not-base64!"}},
		{"configuration": map[string]any{"env": map[string]any{"TOKEN": true}}},
		{"configuration": func() {}},
	} {
		c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			t.Error("invalid patch reached transport")
			return jsonResponse(500, ""), nil
		})})
		got, err := c.PatchConnectorInstance(t.Context(), "worker", patch)
		require.Error(t, err)
		require.Nil(t, got)
	}
}

func TestListResponsesPreserveValidEmptyPages(t *testing.T) {
	c := New("https://stack.example", responseClient(200, emptyPage))
	connectors, err := c.ListConnectors(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connectors.Items)
	require.False(t, connectors.HasMore)
	versions, err := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
	require.NoError(t, err)
	require.Empty(t, versions.Items)
	instances, err := c.ListConnectorInstances(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
}

func TestGivenPublicServer_WhenLifecycleMethodsRun_ThenEveryRouteUsesHTTP(t *testing.T) {
	routes := map[string]struct {
		status int
		body   string
	}{
		"GET /api/connectivity/connectors":                        {200, `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"name":"stripe","latestVersion":"v1.0.0"}]}}`},
		"GET /api/connectivity/connectors/_facets":                {200, `{"total":1,"facets":{"provider":{"psp":1}}}`},
		"GET /api/connectivity/_query/capabilities":               {200, `{"resources":{"connectors":{"name":{"operators":["$match"]}}}}`},
		"GET /api/connectivity/connectors/stripe":                 {200, `{"name":"stripe"}`},
		"GET /api/connectivity/connectors/stripe/versions":        {200, `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"version":"v1.0.0"}]}}`},
		"GET /api/connectivity/connectors/stripe/versions/latest": {200, `{"version":"v1.0.0","configSchema":{"type":"object"}}`},
		"GET /api/connectivity/connector-instances":               {200, emptyPage},
		"POST /api/connectivity/connector-instances":              {201, instanceResponse},
		"GET /api/connectivity/connector-instances/worker":        {200, instanceResponse},
		"PUT /api/connectivity/connector-instances/worker":        {200, instanceResponse},
		"PATCH /api/connectivity/connector-instances/worker":      {200, instanceResponse},
		"DELETE /api/connectivity/connector-instances/worker":     {204, ""},
	}
	seen := make(chan string, len(routes))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := req.Method + " " + req.URL.EscapedPath()
		route, ok := routes[key]
		if !ok {
			http.NotFound(w, req)
			return
		}
		seen <- key
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(route.status)
		_, _ = io.WriteString(w, route.body)
	}))
	t.Cleanup(server.Close)
	c := New(server.URL, server.Client())
	connectors, err := c.ListConnectors(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Equal(t, "stripe", connectors.Items[0].Name)
	require.Equal(t, "v1.0.0", *connectors.Items[0].LatestVersion)
	facets, err := c.GetConnectorFacets(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, int64(1), facets.Facets["provider"]["psp"])
	capabilities, err := c.GetQueryCapabilities(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"$match"}, capabilities.Resources[ResourceConnectors]["name"].Operators)
	connector, err := c.GetConnector(t.Context(), "stripe")
	require.NoError(t, err)
	require.Equal(t, "stripe", connector.Name)
	versions, err := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", versions.Items[0].Version)
	version, err := c.GetConnectorVersion(t.Context(), "stripe", VersionAliasLatest)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", version.Version)
	require.Equal(t, "object", version.ConfigSchema["type"])
	instances, err := c.ListConnectorInstances(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
	configuration := ConnectorInstanceConfiguration{Connector: "stripe", Ledger: "main", Version: pointer("v1.0.0")}
	created, err := c.CreateConnectorInstance(t.Context(), ConnectorInstanceCreate{Name: "worker", Configuration: configuration})
	require.NoError(t, err)
	require.Equal(t, "worker", created.Name)
	fetched, err := c.GetConnectorInstance(t.Context(), "worker")
	require.NoError(t, err)
	require.Equal(t, "worker", fetched.Name)
	replaced, err := c.ReplaceConnectorInstance(t.Context(), "worker", ConnectorInstanceUpdate{Configuration: configuration})
	require.NoError(t, err)
	require.Equal(t, "worker", replaced.Name)
	patched, err := c.PatchConnectorInstance(t.Context(), "worker", ConnectorInstancePatch{"configuration": map[string]any{"pollInterval": "1m"}})
	require.NoError(t, err)
	require.Equal(t, "worker", patched.Name)
	require.NoError(t, c.DeleteConnectorInstance(t.Context(), "worker"))
	require.Len(t, seen, len(routes))
	for range len(routes) {
		delete(routes, <-seen)
	}
	require.Empty(t, routes)
}
