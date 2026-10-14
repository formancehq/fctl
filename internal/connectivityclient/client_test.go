package connectivityclient

import (
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

	got, err := New("https://stack.example/base", httpClient).ListConnectors(t.Context(), ListOptions{
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
				"description":"Payment provider","imageUrl":"https://example.test/stripe.svg","catalog":"providers",
				"branding":{"displayName":"Stripe, Inc.","accentColor":"#635bff","logoSvg":"<svg/>","logoSvgDark":"<svg id=\"dark\"/>"}
			}
		}]}}`), nil
	})}

	got, err := New("https://stack.example", httpClient).ListConnectors(t.Context(), ListOptions{})

	require.NoError(t, err)
	spec := got.Items[0].Spec
	require.Equal(t, "Payments for the internet", *spec.Tagline)
	require.Equal(t, "v2.1.0", *spec.LatestVersion)
	require.Equal(t, []string{"provider:psp"}, spec.Tags)
	require.Equal(t, "Stripe, Inc.", *spec.Branding.DisplayName)
	require.Equal(t, "#635bff", *spec.Branding.AccentColor)
	require.Equal(t, "<svg/>", *spec.Branding.LogoSvg)
	require.Equal(t, `<svg id="dark"/>`, *spec.Branding.LogoSvgDark)
	require.Equal(t, "Payment provider", *spec.Description)
	require.Equal(t, "https://example.test/stripe.svg", *spec.ImageURL)
	require.Equal(t, "providers", *spec.Catalog)
}

func TestClientMethodsRespectHTTPContracts(t *testing.T) {
	connectorName := "stripe/primary"
	instanceName := "worker/one"
	version := "v1.0.0"

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
				_, err := client.ListConnectors(t.Context(), ListOptions{Query: `{"$match":{"catalog":"ee"}}`, PageSize: 5, Cursor: "after"})
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
				_, err := client.GetConnectorFacets(t.Context(), `{"$match":{"catalog":"ee"}}`)
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
				_, err := client.GetQueryCapabilities(t.Context())
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
				_, err := client.GetConnector(t.Context(), connectorName)
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
				_, err := client.ListConnectorVersions(t.Context(), connectorName, ListOptions{PageSize: 100, Cursor: "after"})
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
				_, err := client.GetConnectorVersion(t.Context(), connectorName, "v1.0.0-rc.1")
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
				_, err := client.GetConnectorVersion(t.Context(), connectorName, VersionAliasLatest)
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
				_, err := client.ListConnectorInstances(t.Context(), ListOptions{Query: `{"$match":{"connector":"stripe"}}`, PageSize: 5, Cursor: "after"})
				return err
			},
		},
		{
			name:            "creates a connector instance",
			wantMethod:      http.MethodPost,
			wantPath:        "/base/api/connectivity/connectorinstances",
			wantContentType: "application/json",
			wantBody:        `{"name":"worker","spec":{"connectorRef":{"name":"stripe"},"version":"v1.0.0","ledger":"ledger"}}`,
			responseStatus:  http.StatusCreated,
			responseBody:    `{"metadata":{"name":"worker"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.CreateConnectorInstance(t.Context(), ConnectorInstanceCreate{Name: "worker", Spec: ConnectorInstanceSpec{ConnectorRef: &ObjectRef{Name: "stripe"}, Version: &version, Ledger: "ledger"}})
				return err
			},
		},
		{
			name:           "gets a connector instance using an escaped name",
			wantMethod:     http.MethodGet,
			wantPath:       "/base/api/connectivity/connectorinstances/worker%2Fone",
			responseStatus: http.StatusOK,
			responseBody:   `{"metadata":{"name":"worker/one"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.GetConnectorInstance(t.Context(), instanceName)
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
			responseBody:    `{"metadata":{"name":"worker/one"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"ledger"}}`,
			call: func(client Client) error {
				_, err := client.PatchConnectorInstance(t.Context(), instanceName, ConnectorInstancePatch{"spec": map[string]any{"pollInterval": "1m"}})
				return err
			},
		},
		{
			name:           "deletes a connector instance",
			wantMethod:     http.MethodDelete,
			wantPath:       "/base/api/connectivity/connectorinstances/worker%2Fone",
			responseStatus: http.StatusNoContent,
			call: func(client Client) error {
				return client.DeleteConnectorInstance(t.Context(), instanceName)
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

	_, err := New("https://stack.example", httpClient).ListConnectors(t.Context(), ListOptions{})

	require.NoError(t, err)
	require.Empty(t, seen.URL.RawQuery)
}

func TestGetConnectorFacetsOmitsEmptyQuery(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"total":0,"facets":{}}`), nil
	})}

	got, err := New("https://stack.example", httpClient).GetConnectorFacets(t.Context(), "")

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

	got, err := New("https://stack.example", httpClient).GetQueryCapabilities(t.Context())

	require.NoError(t, err)
	require.Equal(t, []string{"$match", "$in", "$exists"}, got.Resources[ResourceConnectors]["tags"].Operators)
	require.Equal(t, []string{"stable", "rc", "beta", "alpha"}, got.Resources[ResourceConnectorInstances]["channel"].Enum)
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"code":"invalid_query","message":"key not allowed","details":{"key":"nope"}}`), nil
	})}

	_, err := New("https://stack.example", httpClient).GetConnector(t.Context(), "missing")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, "invalid_query", apiErr.Code)
	require.Equal(t, "key not allowed", apiErr.Message)
	require.Equal(t, map[string]any{"key": "nope"}, apiErr.Details)
}

func TestClientReturnsAPIErrorForEmptyUnauthorizedResponse(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		response := jsonResponse(http.StatusUnauthorized, "")
		response.Status = "401 Unauthorized"
		return response, nil
	})}

	_, err := New("https://stack.example", httpClient).ListConnectorInstances(t.Context(), ListOptions{})

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Equal(t, "401 Unauthorized", apiErr.Message)
	require.Empty(t, apiErr.Code)
	require.Nil(t, apiErr.Details)
}

// Configuration assembly helpers must be flattened into the tenant wire spec.
func TestPatchConnectorInstanceSendsAPIShapedConfig(t *testing.T) {
	var seenBody []byte
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		seenBody, err = io.ReadAll(req.Body)
		require.NoError(t, err)
		return jsonResponse(http.StatusOK, `{"metadata":{"name":"worker"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"main"}}`), nil
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

	_, err := New("https://stack.example", httpClient).PatchConnectorInstance(t.Context(), "worker", ConnectorInstancePatch{
		"spec": map[string]any{
			"env":          config.Env,
			"files":        config.Files,
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
				{"path": "/etc/plugin/config.json", "configMapRef": {"name": "connector-config", "key": "config.json"}}
			],
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
				return jsonResponse(http.StatusOK, `{"metadata":{"name":"worker/one"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"main"}}`), nil
			})}

			_, err := New("https://stack.example/base", httpClient).PatchConnectorInstance(
				t.Context(),
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

// A stored legacy instance without connectorRef must not fail the whole page.
func TestListConnectorInstancesDecodesAnInstanceWithNoConnector(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"cursor":{"pageSize":15,"hasMore":false,"data":[
			{"metadata":{"name":"with-connector"},"spec":{"connectorRef":{"name":"stripe"},"ledger":"main"}},
			{"metadata":{"name":"legacy"},"spec":{"ledger":"ops"}},
			{"metadata":{"name":"null-ref"},"spec":{"connectorRef":null,"ledger":"ops"}}
		]}}`), nil
	})}

	list, err := New("https://stack.example", httpClient).ListConnectorInstances(t.Context(), ListOptions{})

	require.NoError(t, err)
	require.Len(t, list.Items, 3)
	require.Equal(t, "stripe", list.Items[0].Spec.ConnectorName())
	require.Empty(t, list.Items[1].Spec.ConnectorName())
	require.Equal(t, "ops", list.Items[1].Spec.Ledger)
	require.Empty(t, list.Items[2].Spec.ConnectorName())
	require.Nil(t, list.Items[2].Spec.ConnectorRef)
}

func TestGetConnectorInstanceDecodesDesiredSuspensionAndObservedProvenance(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"metadata":{"name":"worker"},
			"spec":{"connectorRef":{"name":"stripe"},"ledger":"main","suspend":false},
			"status":{"suspendedBy":[
				{"kind":"ConnectorInstance","name":"worker","field":"spec.suspend"},
				{"kind":"Policy","name":"maintenance","field":"spec.mutate.suspend"}
			]}
		}`), nil
	})}

	instance, err := New("https://stack.example", httpClient).GetConnectorInstance(t.Context(), "worker")

	require.NoError(t, err)
	require.NotNil(t, instance.Spec.Suspend)
	require.False(t, *instance.Spec.Suspend)
	require.Equal(t, []SuspensionSource{
		{Kind: "ConnectorInstance", Name: "worker", Field: "spec.suspend"},
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
				_, err := client.GetConnector(t.Context(), "stripe")
				return err
			},
		},
		{
			name: "empty connector object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetConnector(t.Context(), "stripe")
				return err
			},
		},
		{
			name: "empty connector list object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name: "empty connector version list object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
				return err
			},
		},
		{
			name: "empty connector instance list object",
			body: "null",
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name: "empty facet distribution object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetConnectorFacets(t.Context(), "")
				return err
			},
		},
		{
			name: "empty query capabilities object",
			body: `{}`,
			call: func(client Client) error {
				_, err := client.GetQueryCapabilities(t.Context())
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
				_, err := client.GetConnector(t.Context(), "stripe")
				return err
			},
		},
		{
			name:   "whitespace-only connector object",
			status: http.StatusOK,
			body:   "{ \n\t }",
			call: func(client Client) error {
				_, err := client.GetConnector(t.Context(), "stripe")
				return err
			},
		},
		{
			name:   "connector list missing cursor",
			status: http.StatusOK,
			body:   `{"data":[]}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector list has null data",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":null}}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector instance list has null data",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false}}`,
			call: func(client Client) error {
				_, err := client.ListConnectorInstances(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"displayName":"Stripe"}}`,
			call: func(client Client) error {
				_, err := client.GetConnector(t.Context(), "stripe")
				return err
			},
		},
		{
			name:   "connector list item missing metadata name",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"metadata":{},"spec":{}}]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectors(t.Context(), ListOptions{})
				return err
			},
		},
		{
			name:   "connector version missing image",
			status: http.StatusOK,
			body:   `{"version":"v1.0.0"}`,
			call: func(client Client) error {
				_, err := client.GetConnectorVersion(t.Context(), "stripe", "v1.0.0")
				return err
			},
		},
		{
			name:   "connector version list item missing version",
			status: http.StatusOK,
			body:   `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"image":"registry/stripe:v1.0.0"}]}}`,
			call: func(client Client) error {
				_, err := client.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
				return err
			},
		},
		{
			name:   "connector instance missing metadata name",
			status: http.StatusOK,
			body:   `{"metadata":{},"spec":{"connectorRef":{"name":"stripe"},"ledger":"main"}}`,
			call: func(client Client) error {
				_, err := client.GetConnectorInstance(t.Context(), "worker")
				return err
			},
		},
		// A missing spec.connectorRef or spec.ledger is deliberately absent from
		// this table: neither is structurally invalid on a response. See
		// TestListConnectorInstancesDecodesAnInstanceWithNoConnector.
		{
			name:   "facet distribution missing facets",
			status: http.StatusOK,
			body:   `{"total":3}`,
			call: func(client Client) error {
				_, err := client.GetConnectorFacets(t.Context(), "")
				return err
			},
		},
		{
			name:   "query capabilities missing resources",
			status: http.StatusOK,
			body:   `{"resources":null}`,
			call: func(client Client) error {
				_, err := client.GetQueryCapabilities(t.Context())
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

	connectors, err := client.ListConnectors(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connectors.Items)
	require.False(t, connectors.HasMore)
	require.Empty(t, connectors.Next)
	versions, err := client.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
	require.NoError(t, err)
	require.Empty(t, versions.Items)
	instances, err := client.ListConnectorInstances(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, instances.Items)
}
