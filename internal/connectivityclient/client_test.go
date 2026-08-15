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
		return jsonResponse(200, `{"cursor":{"pageSize":25,"hasMore":true,"next":"next-cursor","data":[{"metadata":{"name":"stripe"},"spec":{"displayName":"Stripe"}}]}}`), nil
	})}

	got, err := New("https://stack.example/base", httpClient).ListConnectors(context.Background(), ListOptions{
		PageSize: 25,
		Cursor:   "opaque",
		Query:    `{"$match":{"catalog":"ee"}}`,
	})

	require.NoError(t, err)
	require.Equal(t, "/base/api/connectivity/connectors", seen.URL.Path)
	require.Equal(t, "25", seen.URL.Query().Get("pageSize"))
	require.Equal(t, "opaque", seen.URL.Query().Get("cursor"))
	require.Equal(t, `{"$match":{"catalog":"ee"}}`, seen.URL.Query().Get("query"))
	require.Equal(t, "stripe", *got.Items[0].Metadata.Name)
	require.Equal(t, int32(25), got.PageSize)
	require.True(t, got.HasMore)
	require.Equal(t, "next-cursor", got.Next)
}

func TestListConnectorsDecodesCatalogueIdentity(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"cursor":{"pageSize":15,"hasMore":false,"data":[{
			"metadata":{"name":"stripe"},
			"spec":{
				"displayName":"Stripe",
				"tagline":"Payments for the internet",
				"latestVersion":"v2.1.0",
				"tags":["provider:psp"],
				"branding":{"displayName":"Stripe, Inc.","accentColor":"#635bff"}
			}
		}]}}`), nil
	})}

	got, err := New("https://stack.example", httpClient).ListConnectors(context.Background(), ListOptions{})

	require.NoError(t, err)
	spec := got.Items[0].Spec
	require.Equal(t, "Payments for the internet", *spec.Tagline)
	require.Equal(t, "v2.1.0", *spec.LatestVersion)
	require.Equal(t, []string{"provider:psp"}, spec.Tags)
	require.Equal(t, "Stripe, Inc.", *spec.Branding.DisplayName)
	require.Equal(t, "#635bff", *spec.Branding.AccentColor)
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
			wantQuery:      map[string]string{"query": `{"$match":{"catalog":"ee"}}`, "pageSize": "5", "cursor": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"cursor":{"pageSize":5,"hasMore":false,"data":[]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{Query: `{"$match":{"catalog":"ee"}}`, PageSize: 5, Cursor: "after"})
				return err
			},
		},
		{
			name:           "gets the connector facet distribution",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/_facets",
			wantQuery:      map[string]string{"query": `{"$match":{"catalog":"ee"}}`},
			responseStatus: http.StatusOK,
			responseBody:   `{"total":6,"facets":{"provider":{"psp":6}}}`,
			call: func(client Client) error {
				_, err := client.GetConnectorFacets(context.Background(), `{"$match":{"catalog":"ee"}}`)
				return err
			},
		},
		{
			name:           "gets the query capabilities",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/_query/capabilities",
			responseStatus: http.StatusOK,
			responseBody:   `{"resources":{"connectors":{"name":{"operators":["$match"]}}}}`,
			call: func(client Client) error {
				_, err := client.GetQueryCapabilities(context.Background())
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
			name:           "lists connector versions with pagination",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/stripe%2Fprimary/versions",
			wantQuery:      map[string]string{"pageSize": "100", "cursor": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"cursor":{"pageSize":100,"hasMore":false,"data":[{"version":"v1.0.0","image":"registry/stripe:v1.0.0"}]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(context.Background(), connectorName, ListOptions{PageSize: 100, Cursor: "after"})
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
			name:           "resolves a version alias in the version slot",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectors/stripe%2Fprimary/versions/latest",
			responseStatus: http.StatusOK,
			responseBody:   `{"version":"v2.0.0","image":"registry/stripe:v2.0.0"}`,
			call: func(client Client) error {
				_, err := client.GetConnectorVersion(context.Background(), connectorName, VersionAliasLatest)
				return err
			},
		},
		{
			name:           "lists connector instances",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectorinstances",
			wantQuery:      map[string]string{"query": `{"$match":{"connector":"stripe"}}`, "pageSize": "5", "cursor": "after"},
			responseStatus: http.StatusOK,
			responseBody:   `{"cursor":{"pageSize":5,"hasMore":false,"data":[]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(context.Background(), ListOptions{Query: `{"$match":{"connector":"stripe"}}`, PageSize: 5, Cursor: "after"})
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

func TestListRequestsOmitUnsetOptions(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"cursor":{"pageSize":15,"hasMore":false,"data":[]}}`), nil
	})}

	_, err := New("https://stack.example", httpClient).ListConnectors(context.Background(), ListOptions{})

	require.NoError(t, err)
	require.Empty(t, seen.URL.RawQuery)
}

func TestGetConnectorFacetsOmitsEmptyQuery(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"total":0,"facets":{}}`), nil
	})}

	got, err := New("https://stack.example", httpClient).GetConnectorFacets(context.Background(), "")

	require.NoError(t, err)
	require.Empty(t, seen.URL.RawQuery)
	require.Zero(t, got.Total)
	require.Empty(t, got.Facets)
}

func TestGetQueryCapabilitiesDecodesResources(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"resources":{
			"connectors":{"tags":{"operators":["$match","$in","$exists"]}},
			"connectorinstances":{"channel":{"operators":["$match","$in","$exists"],"enum":["stable","rc","beta","alpha"]}}
		}}`), nil
	})}

	got, err := New("https://stack.example", httpClient).GetQueryCapabilities(context.Background())

	require.NoError(t, err)
	require.Equal(t, []string{"$match", "$in", "$exists"}, got.Resources[ResourceConnectors]["tags"].Operators)
	require.Equal(t, []string{"stable", "rc", "beta", "alpha"}, got.Resources[ResourceConnectorInstances]["channel"].Enum)
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"code":"invalid_query","message":"key not allowed","details":{"key":"nope"}}`), nil
	})}

	_, err := New("https://stack.example", httpClient).GetConnector(context.Background(), "missing")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, "invalid_query", apiErr.Code)
	require.Equal(t, "key not allowed", apiErr.Message)
	require.Equal(t, map[string]any{"key": "nope"}, apiErr.Details)
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

func TestPatchConnectorInstanceSendsExactSuspensionBoolean(t *testing.T) {
	tests := []struct {
		name    string
		suspend bool
		want    string
	}{
		{name: "suspend", suspend: true, want: `{"spec":{"suspend":true}}`},
		{name: "explicit resume", suspend: false, want: `{"spec":{"suspend":false}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var seenMethod, seenPath, seenContentType string
			var seenBody []byte
			httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				seenMethod = req.Method
				seenPath = req.URL.EscapedPath()
				seenContentType = req.Header.Get("Content-Type")
				var err error
				seenBody, err = io.ReadAll(req.Body)
				require.NoError(t, err)
				return jsonResponse(http.StatusOK, `{"metadata":{"name":"worker/one"},"spec":{"connector":"stripe","ledger":"main"}}`), nil
			})}

			_, err := New("https://stack.example/base", httpClient).PatchConnectorInstance(
				context.Background(),
				"worker/one",
				ConnectorInstancePatch{"spec": map[string]any{"suspend": test.suspend}},
			)

			require.NoError(t, err)
			require.Equal(t, http.MethodPatch, seenMethod)
			require.Equal(t, "/base/api/connectivity/connectorinstances/worker%2Fone", seenPath)
			require.Equal(t, "application/merge-patch+json", seenContentType)
			require.Equal(t, test.want, string(seenBody))
		})
	}
}

// An instance pinned to spec.image on the CR references no Connector, so the
// API returns it with an empty spec.connector. One such row used to fail the
// decode of the whole page -- `connectorinstances list` reported
// "cursor.data[N]: connector instance spec.connector is required" on a
// response the server had answered 200.
func TestListConnectorInstancesDecodesAnInstanceWithNoConnector(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"cursor":{"pageSize":15,"hasMore":false,"data":[
			{"metadata":{"name":"with-connector"},"spec":{"connector":"stripe","ledger":"main"}},
			{"metadata":{"name":"image-pinned"},"spec":{"ledger":"ops"}}
		]}}`), nil
	})}

	list, err := New("https://stack.example", httpClient).ListConnectorInstances(context.Background(), ListOptions{})

	require.NoError(t, err)
	require.Len(t, list.Items, 2)
	require.Equal(t, "stripe", list.Items[0].Spec.Connector)
	require.Empty(t, list.Items[1].Spec.Connector)
	require.Equal(t, "ops", list.Items[1].Spec.Ledger)
}

func TestGetConnectorInstanceDecodesDesiredSuspensionAndObservedProvenance(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"metadata":{"name":"worker"},
			"spec":{"connector":"stripe","ledger":"main","suspend":false,"replicas":0},
			"status":{"suspendedBy":[
				{"kind":"ConnectorInstance","name":"worker","field":"spec.replicas"},
				{"kind":"Policy","name":"maintenance","field":"spec.mutate.suspend"}
			]}
		}`), nil
	})}

	instance, err := New("https://stack.example", httpClient).GetConnectorInstance(context.Background(), "worker")

	require.NoError(t, err)
	require.NotNil(t, instance.Spec.Suspend)
	require.False(t, *instance.Spec.Suspend)
	require.NotNil(t, instance.Spec.Replicas)
	require.Equal(t, int32(0), *instance.Spec.Replicas)
	require.Equal(t, []SuspensionSource{
		{Kind: "ConnectorInstance", Name: "worker", Field: "spec.replicas"},
		{Kind: "Policy", Name: "maintenance", Field: "spec.mutate.suspend"},
	}, instance.Status.SuspendedBy)
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
				_, err := client.ListConnectorVersions(context.Background(), "stripe", ListOptions{})
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
		{
			name: "empty facet distribution object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetConnectorFacets(context.Background(), "")
				return err
			},
		},
		{
			name: "empty query capabilities object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetQueryCapabilities(context.Background())
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
			name:   "connector list missing cursor",
			status: http.StatusOK,
			body:   `{"data":[]}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector list has null data",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":null}}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(context.Background(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector instance list has null data",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false}}`,
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
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"metadata":{},"spec":{}}]}}`,
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
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"image":"registry/stripe:v1.0.0"}]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(context.Background(), "stripe", ListOptions{})
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
		// A missing spec.connector or spec.ledger is deliberately absent from
		// this table: neither is structurally invalid on a response. See
		// TestListConnectorInstancesDecodesAnInstanceWithNoConnector.
		{
			name:   "facet distribution missing facets",
			status: http.StatusOK,
			body:   `{"total":3}`,
			call: func(client Client) error {
				_, err := client.GetConnectorFacets(context.Background(), "")
				return err
			},
		},
		{
			name:   "query capabilities missing resources",
			status: http.StatusOK,
			body:   `{"resources":null}`,
			call: func(client Client) error {
				_, err := client.GetQueryCapabilities(context.Background())
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
		return jsonResponse(http.StatusOK, `{"cursor":{"pageSize":15,"hasMore":false,"data":[]}}`), nil
	})}
	client := New("https://stack.example", httpClient)

	connectors, err := client.ListConnectors(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connectors.Items)
	require.False(t, connectors.HasMore)
	require.Empty(t, connectors.Next)
	versions, err := client.ListConnectorVersions(context.Background(), "stripe", ListOptions{})
	require.NoError(t, err)
	require.Empty(t, versions.Items)
	instances, err := client.ListConnectorInstances(context.Background(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
}
