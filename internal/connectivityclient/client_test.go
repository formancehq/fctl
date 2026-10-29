package connectivityclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func pointer[T any](value T) *T { return &value }

const instanceResponse = `{"name":"worker","configuration":{"connector":"stripe","version":"v1.0.0","ledger":"main"}}`
const emptyPage = `{"cursor":{"pageSize":15,"hasMore":false,"data":[]}}`

func TestClientMethodsRespectHTTPContracts(t *testing.T) {
	configuration := ConnectionConfiguration{Connector: "stripe", Version: pointer("v1.0.0"), Ledger: pointer("main")}
	tests := []struct {
		name, method, path, body, contentType, response string
		status                                          int
		call                                            func(Client) error
	}{
		{name: "connectors", method: http.MethodGet, path: "connectors", response: emptyPage, status: 200, call: func(c Client) error { _, err := c.ListConnectors(t.Context(), ListOptions{}); return err }},
		{name: "facets", method: http.MethodGet, path: "connectors/_facets", response: `{"total":0,"facets":{}}`, status: 200, call: func(c Client) error { _, err := c.GetConnectorFacets(t.Context(), ""); return err }},
		{name: "capabilities", method: http.MethodGet, path: "_query/capabilities", response: `{"resources":{}}`, status: 200, call: func(c Client) error { _, err := c.GetQueryCapabilities(t.Context()); return err }},
		{name: "connector", method: http.MethodGet, path: "connectors/stripe%2Fprimary", response: `{"name":"stripe"}`, status: 200, call: func(c Client) error { _, err := c.GetConnector(t.Context(), "stripe/primary"); return err }},
		{name: "versions", method: http.MethodGet, path: "connectors/stripe/versions", response: emptyPage, status: 200, call: func(c Client) error {
			_, err := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
			return err
		}},
		{name: "version alias", method: http.MethodGet, path: "connectors/stripe/versions/latest", response: `{"version":"v1.0.0"}`, status: 200, call: func(c Client) error {
			_, err := c.GetConnectorVersion(t.Context(), "stripe", VersionAliasLatest)
			return err
		}},
		{name: "instances", method: http.MethodGet, path: "connections", response: emptyPage, status: 200, call: func(c Client) error { _, err := c.ListConnections(t.Context(), ListOptions{}); return err }},
		{name: "get", method: http.MethodGet, path: "connections/worker%2Fone", response: instanceResponse, status: 200, call: func(c Client) error { _, err := c.GetConnection(t.Context(), "worker/one"); return err }},
		{name: "create", method: http.MethodPost, path: "connections", body: `{"name":"worker","configuration":{"connector":"stripe","version":"v1.0.0","ledger":"main"}}`, contentType: "application/json", response: instanceResponse, status: 201, call: func(c Client) error {
			_, err := c.CreateConnection(t.Context(), ConnectionCreate{Name: "worker", Configuration: configuration})
			return err
		}},
		{name: "replace", method: http.MethodPut, path: "connections/worker%2Fone", body: `{"configuration":{"connector":"stripe","version":"v1.0.0","ledger":"main"}}`, contentType: "application/json", response: instanceResponse, status: 200, call: func(c Client) error {
			_, err := c.ReplaceConnection(t.Context(), "worker/one", ConnectionUpdate{Configuration: configuration})
			return err
		}},
		{name: "patch", method: http.MethodPatch, path: "connections/worker%2Fone", body: `{"configuration":{"suspend":false}}`, contentType: "application/merge-patch+json", response: instanceResponse, status: 200, call: func(c Client) error {
			_, err := c.PatchConnection(t.Context(), "worker/one", ConnectionPatch{"configuration": map[string]any{"suspend": false}})
			return err
		}},
		{name: "delete", method: http.MethodDelete, path: "connections/worker%2Fone", status: 204, call: func(c Client) error { return c.DeleteConnection(t.Context(), "worker/one") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen *http.Request
			var body []byte
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				seen = req
				if req.Body != nil {
					var err error
					body, err = io.ReadAll(req.Body)
					require.NoError(t, err)
				}
				return jsonResponse(tt.status, tt.response), nil
			})
			require.NoError(t, tt.call(New("https://stack.example/base/?discard=true#fragment", &http.Client{Transport: transport})))
			require.Equal(t, tt.method, seen.Method)
			require.Contains(t, seen.Header.Get("User-Agent"), "speakeasy-sdk/go")
			require.Equal(t, "/base/api/connectivity/"+tt.path, seen.URL.EscapedPath())
			require.Empty(t, seen.URL.RawQuery)
			require.Empty(t, seen.URL.Fragment)
			require.Equal(t, "application/json", seen.Header.Get("Accept"))
			require.Equal(t, tt.contentType, seen.Header.Get("Content-Type"))
			if tt.body != "" {
				require.JSONEq(t, tt.body, string(body))
			} else {
				require.Empty(t, body)
			}
		})
	}
}

func TestListOptionsAndCursorArePreserved(t *testing.T) {
	options := ListOptions{PageSize: 25, Cursor: "opaque +/=", Query: `{"$match":{"catalog":"ee"}}`}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "25", req.URL.Query().Get("pageSize"))
		require.Equal(t, options.Cursor, req.URL.Query().Get("cursor"))
		require.Equal(t, options.Query, req.URL.Query().Get("query"))
		return jsonResponse(200, `{"cursor":{"pageSize":25,"hasMore":true,"next":"next+/=","data":[{"name":"stripe"}]}}`), nil
	})
	got, err := New("https://stack.example", &http.Client{Transport: transport}).ListConnectors(t.Context(), options)
	require.NoError(t, err)
	require.Equal(t, "stripe", got.Items[0].Name)
	require.Equal(t, int32(25), got.PageSize)
	require.True(t, got.HasMore)
	require.Equal(t, "next+/=", got.Next)
}

func TestFacetQueryAndResourceCapabilities(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "_facets") {
			require.Equal(t, `{"$match":{"tags":"provider:psp"}}`, req.URL.Query().Get("query"))
			return jsonResponse(200, `{"total":1,"facets":{"provider":{"psp":1}}}`), nil
		}
		return jsonResponse(200, `{"resources":{"connections":{"channel":{"operators":["$match","$in"],"enum":["stable","rc","beta","alpha"]}},"connectorversions":{"version":{"operators":["$match"]}}}}`), nil
	})
	c := New("https://stack.example", &http.Client{Transport: transport})
	facets, err := c.GetConnectorFacets(t.Context(), `{"$match":{"tags":"provider:psp"}}`)
	require.NoError(t, err)
	require.Equal(t, int64(1), facets.Total)
	require.Equal(t, int64(1), facets.Facets["provider"]["psp"])
	caps, err := c.GetQueryCapabilities(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"stable", "rc", "beta", "alpha"}, caps.Resources[ResourceConnections]["channel"].Enum)
	require.Contains(t, caps.Resources, ResourceConnectorVersions)
}

func TestClientReturnsAPIError(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
	}{
		{"structured", `{"code":"ledger_immutable","message":"ledger cannot change","details":{"field":"ledger"}}`, 422},
		{"empty unauthorized", "", 401}, {"non-JSON unauthorized", "private upstream diagnostic", 401}, {"malformed", "not JSON", 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(tt.status, tt.body), nil })})
			got, err := c.GetConnection(t.Context(), "worker")
			require.Nil(t, got)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tt.status, apiErr.StatusCode)
			require.Contains(t, err.Error(), "connectivity API error")
			if tt.name == "structured" {
				require.Equal(t, "ledger_immutable", apiErr.Code)
				require.Equal(t, "ledger cannot change", apiErr.Message)
				require.Equal(t, map[string]any{"field": "ledger"}, apiErr.Details)
			} else {
				require.Equal(t, http.StatusText(tt.status), apiErr.Message)
				require.Empty(t, apiErr.Code)
			}
		})
	}
}

func TestClientPropagatesTransportAndContextErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })})
	_, err := c.GetConnector(ctx, "stripe")
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "send connectivity request")
}

func TestClientRejectsInvalidEndpoints(t *testing.T) {
	for _, uri := range []string{"%", "relative", "/", "https://"} {
		_, err := New(uri, nil).GetConnector(t.Context(), "stripe")
		require.ErrorContains(t, err, "parse stack URI")
	}
}

func TestClientRejectsInvalidPublicResponses(t *testing.T) {
	tests := []struct {
		name, body string
		call       func(Client) error
	}{
		{"malformed", `{`, func(c Client) error { _, e := c.GetConnector(t.Context(), "stripe"); return e }},
		{"empty body", "", func(c Client) error { _, e := c.GetConnector(t.Context(), "stripe"); return e }},
		{"root array", `[]`, func(c Client) error { _, e := c.GetConnector(t.Context(), "stripe"); return e }},
		{"missing name", `{}`, func(c Client) error { _, e := c.GetConnector(t.Context(), "stripe"); return e }},
		{"old connector", `{"metadata":{"name":"stripe"},"spec":{}}`, func(c Client) error { _, e := c.GetConnector(t.Context(), "stripe"); return e }},
		{"missing cursor", `{}`, func(c Client) error { _, e := c.ListConnectors(t.Context(), ListOptions{}); return e }},
		{"null cursor", `{"cursor":null}`, func(c Client) error { _, e := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{}); return e }},
		{"null data", `{"cursor":{"data":null}}`, func(c Client) error { _, e := c.ListConnections(t.Context(), ListOptions{}); return e }},
		{"bad list connector", `{"cursor":{"data":[{"name":"stripe"},{}]}}`, func(c Client) error { _, e := c.ListConnectors(t.Context(), ListOptions{}); return e }},
		{"missing version", `{}`, func(c Client) error { _, e := c.GetConnectorVersion(t.Context(), "stripe", "latest"); return e }},
		{"bad list version", `{"cursor":{"data":[{}]}}`, func(c Client) error { _, e := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{}); return e }},
		{"missing facets", `{"total":0}`, func(c Client) error { _, e := c.GetConnectorFacets(t.Context(), ""); return e }},
		{"missing resources", `{"resources":null}`, func(c Client) error { _, e := c.GetQueryCapabilities(t.Context()); return e }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(200, tt.body), nil })})
			require.ErrorContains(t, tt.call(c), "decode connectivity response")
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func TestDecodeResponseReadError(t *testing.T) {
	require.ErrorContains(t, decodeResponse(failingReader{}, &Connector{}, true), "read connectivity response")
	var connector Connector
	require.ErrorContains(t, decodeResponse(strings.NewReader(`{"name":42}`), &connector, true), "decode connectivity response")
}

func TestEveryListMethodPreservesFiltersPaginationAndEscapedConnector(t *testing.T) {
	options := ListOptions{PageSize: 25, Cursor: "opaque +/=", Query: `{"$match":{"name":"stripe"}}`}
	tests := []struct {
		name, path, item string
		call             func(Client) (int32, bool, string, int, error)
	}{
		{"connectors", "connectors", `{"name":"stripe"}`, func(c Client) (int32, bool, string, int, error) {
			page, err := c.ListConnectors(t.Context(), options)
			if err != nil {
				return 0, false, "", 0, err
			}
			return page.PageSize, page.HasMore, page.Next, len(page.Items), nil
		}},
		{"versions", "connectors/stripe%2Fprimary/versions", `{"version":"v1.0.0"}`, func(c Client) (int32, bool, string, int, error) {
			page, err := c.ListConnectorVersions(t.Context(), "stripe/primary", options)
			if err != nil {
				return 0, false, "", 0, err
			}
			return page.PageSize, page.HasMore, page.Next, len(page.Items), nil
		}},
		{"instances", "connections", instanceResponse, func(c Client) (int32, bool, string, int, error) {
			page, err := c.ListConnections(t.Context(), options)
			if err != nil {
				return 0, false, "", 0, err
			}
			return page.PageSize, page.HasMore, page.Next, len(page.Items), nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "/base/api/connectivity/"+tt.path, req.URL.EscapedPath())
				require.Equal(t, "25", req.URL.Query().Get("pageSize"))
				require.Equal(t, options.Cursor, req.URL.Query().Get("cursor"))
				require.Equal(t, options.Query, req.URL.Query().Get("query"))
				return jsonResponse(200, `{"cursor":{"pageSize":25,"hasMore":true,"next":"next+/=","data":[`+tt.item+`]}}`), nil
			})
			pageSize, hasMore, next, count, err := tt.call(New("https://stack.example/base", &http.Client{Transport: transport}))
			require.NoError(t, err)
			require.Equal(t, int32(25), pageSize)
			require.True(t, hasMore)
			require.Equal(t, "next+/=", next)
			require.Equal(t, 1, count)
		})
	}
}

func TestVersionLookupEscapesConnectorAndVersionSegments(t *testing.T) {
	for _, version := range []string{"v1.0.0-rc.1", VersionAliasLatest, "stable"} {
		t.Run(version, func(t *testing.T) {
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "/base/api/connectivity/connectors/stripe%2Fprimary/versions/"+version, req.URL.EscapedPath())
				return jsonResponse(200, `{"version":"v1.0.0"}`), nil
			})
			got, err := New("https://stack.example/base", &http.Client{Transport: transport}).GetConnectorVersion(t.Context(), "stripe/primary", version)
			require.NoError(t, err)
			require.Equal(t, "v1.0.0", got.Version)
		})
	}
}

func TestEmptyFacetsAndCapabilityOperatorsArePreserved(t *testing.T) {
	var seenQuery string
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "_facets") {
			seenQuery = req.URL.RawQuery
			return jsonResponse(200, `{"total":0,"facets":{}}`), nil
		}
		return jsonResponse(200, `{"resources":{"connectors":{"tags":{"operators":["$match","$in","$exists"]}},"connections":{"channel":{"operators":["$match","$in","$exists"],"enum":["stable","rc","beta","alpha"]}}}}`), nil
	})
	c := New("https://stack.example", &http.Client{Transport: transport})
	facets, err := c.GetConnectorFacets(t.Context(), "")
	require.NoError(t, err)
	require.Empty(t, seenQuery)
	require.Zero(t, facets.Total)
	require.NotNil(t, facets.Facets)
	require.Empty(t, facets.Facets)
	caps, err := c.GetQueryCapabilities(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"$match", "$in", "$exists"}, caps.Resources[ResourceConnectors]["tags"].Operators)
	require.Equal(t, []string{"$match", "$in", "$exists"}, caps.Resources[ResourceConnections]["channel"].Operators)
	require.Equal(t, []string{"stable", "rc", "beta", "alpha"}, caps.Resources[ResourceConnections]["channel"].Enum)
}

func TestListConnectorsDecodesPublishedCatalogueFields(t *testing.T) {
	c := New("https://stack.example", responseClient(200, `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"name":"stripe","displayName":"Stripe","description":"Payments","catalog":"providers","tagline":"Simple payments","latestVersion":"v2.1.0","tags":["provider:psp"],"branding":{"displayName":"Stripe Inc.","accentColor":"#635bff","logoSvg":"<svg/>","logoSvgDark":"<svg id='dark'/>"}}]}}`))
	page, err := c.ListConnectors(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	connector := page.Items[0]
	require.Equal(t, "stripe", connector.Name)
	require.Equal(t, "Stripe", *connector.DisplayName)
	require.Equal(t, "Payments", *connector.Description)
	require.Equal(t, "providers", *connector.Catalog)
	require.Equal(t, "Simple payments", *connector.Tagline)
	require.Equal(t, "v2.1.0", *connector.LatestVersion)
	require.Equal(t, []string{"provider:psp"}, connector.Tags)
	require.Equal(t, "Stripe Inc.", *connector.Branding.DisplayName)
	require.Equal(t, "#635bff", *connector.Branding.AccentColor)
	require.Equal(t, "<svg/>", *connector.Branding.LogoSvg)
	require.Equal(t, "<svg id='dark'/>", *connector.Branding.LogoSvgDark)
}

func TestAllListsRejectMalformedEnvelopes(t *testing.T) {
	tests := []struct {
		name string
		call func(Client) error
	}{
		{"connectors", func(c Client) error { _, err := c.ListConnectors(t.Context(), ListOptions{}); return err }},
		{"versions", func(c Client) error {
			_, err := c.ListConnectorVersions(t.Context(), "stripe", ListOptions{})
			return err
		}},
		{"instances", func(c Client) error { _, err := c.ListConnections(t.Context(), ListOptions{}); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, body := range []string{"null", `{}`, "{ \n\t }", `{"data":[]}`, `{"cursor":null}`, `{"cursor":{"pageSize":15,"hasMore":false}}`, `{"cursor":{"pageSize":15,"hasMore":false,"data":null}}`} {
				require.ErrorContains(t, tt.call(New("https://stack.example", responseClient(200, body))), "decode connectivity response")
			}
		})
	}
}

func TestInvalidInstanceNameReportsListItemIndex(t *testing.T) {
	for _, name := range []string{"", `"name":null,`, `"name":"",`, `"name":"  ",`} {
		body := `{` + name + `"configuration":{"ledger":"main"}}`
		c := New("https://stack.example", responseClient(200, body))
		got, err := c.GetConnection(t.Context(), "worker")
		require.Nil(t, got)
		require.ErrorContains(t, err, "connection name is required")
		c = New("https://stack.example", responseClient(200, `{"cursor":{"data":[`+instanceResponse+`,`+body+`]}}`))
		page, err := c.ListConnections(t.Context(), ListOptions{})
		require.Nil(t, page)
		require.ErrorContains(t, err, "cursor.data[1]: connection name is required")
	}
}

func TestInstanceGETAndListAcceptCurrentOptionalStatusContract(t *testing.T) {
	for _, status := range []string{"", `,"status":null`, `,"status":{"lifecycle":"pending","suspendedBy":[]}`, `,"status":{"appliedVersion":"v1.2.0","lifecycle":"active","suspendedBy":[]}`} {
		body := `{"name":"worker","configuration":{"ledger":"main"}` + status + `}`
		c := New("https://stack.example", responseClient(200, body))
		got, err := c.GetConnection(t.Context(), "worker")
		require.NoError(t, err)
		require.Equal(t, "worker", got.Name)
		c = New("https://stack.example", responseClient(200, `{"cursor":{"data":[`+body+`]}}`))
		page, err := c.ListConnections(t.Context(), ListOptions{})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
	}
}

func TestClientReturnsAPIErrorForEmptyUnauthorizedResponse(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		response := jsonResponse(http.StatusUnauthorized, "")
		response.Status = "401 Unauthorized"
		return response, nil
	})
	_, err := New("https://stack.example", &http.Client{Transport: transport}).ListConnections(t.Context(), ListOptions{})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Equal(t, "401 Unauthorized", apiErr.Message)
	require.Empty(t, apiErr.Code)
	require.Nil(t, apiErr.Details)
}
