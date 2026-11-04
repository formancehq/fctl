package connectors

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
)

type connectorRoundTripper func(*http.Request) (*http.Response, error)

func (f connectorRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func sdkFactory(transport connectorRoundTripper) connectivityinternal.ClientFactory {
	sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: transport}))
	return factoryReturning(sdk.Connectivity.V1)
}

func connectorResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestConnectorCommandsUseGeneratedSDK(t *testing.T) {
	for _, test := range []struct {
		name    string
		command func(connectivityinternal.ClientFactory) *cobra.Command
		path    string
		body    string
		want    string
	}{
		{"list", NewListCommand, "/api/connectivity/connectors", `{"cursor":{"data":[{"name":"stripe"}],"pageSize":15,"hasMore":false}}`, "stripe"},
		{"facets", NewFacetsCommand, "/api/connectivity/connectors/_facets", `{"total":1,"facets":{"catalog":{"ee":1}}}`, "catalog"},
	} {
		t.Run(test.name, func(t *testing.T) {
			factory := sdkFactory(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, request.Method)
				require.Equal(t, test.path, request.URL.Path)
				require.JSONEq(t, `{"$match":{"catalog":"ee"}}`, request.URL.Query().Get("query"))
				require.False(t, request.URL.Query().Has("cursor"))
				return connectorResponse(200, test.body), nil
			})
			output, err := executeCommand(test.command(factory), "--filter", "catalog=ee")
			require.NoError(t, err)
			require.Contains(t, output, test.want)
		})
	}
}

func TestConnectorCommandsRejectGeneratedErrorEnvelopes(t *testing.T) {
	for _, constructor := range []func(connectivityinternal.ClientFactory) *cobra.Command{NewListCommand, NewFacetsCommand, NewShowCommand} {
		command := constructor(nil)
		t.Run(command.Name(), func(t *testing.T) {
			for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest} {
				factory := sdkFactory(func(*http.Request) (*http.Response, error) {
					return connectorResponse(status, `{"code":"invalid_query","message":"invalid filter"}`), nil
				})
				command := constructor(factory)
				command.Flags().String(fctl.OutputFlag, "json", "")
				command.SilenceUsage = true
				command.SilenceErrors = true
				var out bytes.Buffer
				command.SetOut(&out)
				command.SetErr(io.Discard)
				args := []string{}
				if command.Name() == "show" {
					args = append(args, "stripe")
				}
				command.SetArgs(args)
				err := command.Execute()
				require.Error(t, err)
				var apiErr *connectivityinternal.APIError
				require.ErrorAs(t, err, &apiErr)
				require.Equal(t, status, apiErr.StatusCode)
				require.Empty(t, out.String())
			}
		})
	}
}

func TestShowConnectorHandlesGeneratedLatestNotFound(t *testing.T) {
	calls := 0
	factory := sdkFactory(func(request *http.Request) (*http.Response, error) {
		calls++
		switch request.URL.Path {
		case "/api/connectivity/connectors/stripe":
			return connectorResponse(200, `{"name":"stripe"}`), nil
		case "/api/connectivity/connectors/stripe/versions":
			return connectorResponse(200, `{"cursor":{"data":[{"version":"1.0.0"}],"hasMore":false,"pageSize":100}}`), nil
		case "/api/connectivity/connectors/stripe/versions/latest":
			return connectorResponse(404, `{"code":"channel_empty","message":"No validated version"}`), nil
		default:
			t.Fatalf("unexpected SDK route %s", request.URL.Path)
			return nil, nil
		}
	})
	output, err := executeCommand(NewShowCommand(factory), "stripe")
	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Contains(t, output, "No published version.")
}

func TestShowConnectorEscapesNameAsOnePathSegment(t *testing.T) {
	paths := []string{}
	factory := sdkFactory(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.EscapedPath())
		require.Empty(t, request.URL.Query().Get("c"))
		body := `{"name":"a/b?c#d"}`
		if len(paths) == 2 {
			body = `{"cursor":{"data":[],"hasMore":false,"pageSize":100}}`
		}
		return connectorResponse(200, body), nil
	})
	_, err := executeCommand(NewShowCommand(factory), "a/b?c#d")
	require.NoError(t, err)
	require.Equal(t, []string{"/api/connectivity/connectors/a%2Fb%3Fc%23d", "/api/connectivity/connectors/a%2Fb%3Fc%23d/versions"}, paths)
}

func TestConnectorSDKErrorPreservesResponseBody(t *testing.T) {
	factory := sdkFactory(func(*http.Request) (*http.Response, error) {
		response := connectorResponse(http.StatusBadGateway, "upstream diagnostic")
		response.Header.Set("Content-Type", "text/plain")
		return response, nil
	})
	_, err := executeCommand(NewListCommand(factory))
	require.ErrorContains(t, err, "upstream diagnostic")
}

func TestConnectorCommandsValidateGeneratedSuccessPayloads(t *testing.T) {
	for _, test := range []struct {
		name        string
		constructor func(connectivityinternal.ClientFactory) *cobra.Command
		bodies      []string
		emptyBody   string
		wantError   string
	}{
		{"list", NewListCommand, []string{`{}`, `{"cursor":null}`, `{"cursor":{}}`, `{"cursor":{"data":null}}`}, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`, "cursor.data must be an array"},
		{"show", NewShowCommand, []string{`{}`, `{"cursor":null}`, `{"cursor":{}}`, `{"cursor":{"data":null}}`}, `{"cursor":{"data":[],"hasMore":false,"pageSize":100}}`, "cursor.data must be an array"},
		{"facets", NewFacetsCommand, []string{`{}`, `{"total":0}`, `{"total":0,"facets":null}`}, `{"total":0,"facets":{}}`, "facets must be an object"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, body := range append(test.bodies, test.emptyBody) {
				t.Run(body, func(t *testing.T) {
					factory := sdkFactory(func(request *http.Request) (*http.Response, error) {
						if test.name == "show" && request.URL.Path == "/api/connectivity/connectors/stripe" {
							return connectorResponse(200, `{"name":"stripe"}`), nil
						}
						return connectorResponse(200, body), nil
					})
					command := test.constructor(factory)
					command.Flags().String(fctl.OutputFlag, "json", "")
					command.SilenceUsage = true
					command.SilenceErrors = true
					var out bytes.Buffer
					command.SetOut(&out)
					command.SetErr(io.Discard)
					args := []string{}
					if test.name == "show" {
						args = append(args, "stripe")
					}
					command.SetArgs(args)
					err := command.Execute()
					if body == test.emptyBody {
						require.NoError(t, err)
						require.NotEmpty(t, out.String())
					} else {
						require.ErrorContains(t, err, test.wantError)
						require.Empty(t, out.String())
					}
				})
			}
		})
	}
}
