package connectors

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

type connectorRoundTripper func(*http.Request) (*http.Response, error)

func (f connectorRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func sdkWithTransport(transport connectorRoundTripper) *formance.FormanceV1 {
	sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: transport}))
	return sdk.Connectivity.V1
}

func connectorResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestConnectorCommandsUseGeneratedSDK(t *testing.T) {
	for _, test := range []struct {
		name    string
		command func(*formance.FormanceV1) *cobra.Command
		path    string
		body    string
		want    string
	}{
		{"list", testNewListCommand, "/api/connectivity/connectors", `{"cursor":{"data":[{"name":"stripe"}],"pageSize":15,"hasMore":false}}`, "stripe"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := sdkWithTransport(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, request.Method)
				require.Equal(t, test.path, request.URL.Path)
				require.JSONEq(t, `{"$match":{"catalog":"ee"}}`, request.URL.Query().Get("query"))
				require.False(t, request.URL.Query().Has("cursor"))
				return connectorResponse(200, test.body), nil
			})
			output, err := executeCommand(test.command(client), "--filter", "catalog=ee")
			require.NoError(t, err)
			require.Contains(t, output, test.want)
		})
	}
}

func TestConnectorCommandsRejectGeneratedErrorEnvelopes(t *testing.T) {
	for _, constructor := range []func(*formance.FormanceV1) *cobra.Command{testNewListCommand, testNewShowCommand} {
		command := constructor(nil)
		t.Run(command.Name(), func(t *testing.T) {
			for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest} {
				var rawResponse *http.Response
				client := sdkWithTransport(func(*http.Request) (*http.Response, error) {
					rawResponse = connectorResponse(status, `{"errorCode":"invalid_query","errorMessage":"invalid filter"}`)
					return rawResponse, nil
				})
				command := constructor(client)
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
				if status == http.StatusUnauthorized {
					var apiErr *sdkerrors.SDKError
					require.ErrorAs(t, err, &apiErr)
					require.Same(t, apiErr, err)
					require.Equal(t, status, apiErr.StatusCode)
					require.Same(t, rawResponse, apiErr.RawResponse)
					require.JSONEq(t, `{"errorCode":"invalid_query","errorMessage":"invalid filter"}`, apiErr.Body)
				} else {
					var apiErr *connectivity.Error
					require.ErrorAs(t, err, &apiErr)
					require.Same(t, apiErr, err)
					require.Equal(t, "invalid_query", *apiErr.ErrorCode)
					require.Equal(t, "invalid filter", *apiErr.ErrorMessage)
				}
				require.Empty(t, out.String())
			}
		})
	}
}

func TestShowConnectorReturnsGeneratedPublishedVersionNotFound(t *testing.T) {
	calls := 0
	client := sdkWithTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		switch request.URL.Path {
		case "/api/connectivity/connectors/stripe":
			return connectorResponse(200, `{"name":"stripe","latestVersion":"1.0.0"}`), nil
		case "/api/connectivity/connectors/stripe/versions":
			return connectorResponse(200, `{"cursor":{"data":[{"version":"1.0.0"}],"hasMore":false,"pageSize":100}}`), nil
		case "/api/connectivity/connectors/stripe/versions/1.0.0":
			return connectorResponse(404, `{"errorCode":"channel_empty","errorMessage":"No validated version"}`), nil
		default:
			t.Fatalf("unexpected SDK route %s", request.URL.Path)
			return nil, nil
		}
	})
	output, err := executeCommand(testNewShowCommand(client), "stripe")
	var apiErr *connectivity.Error
	require.ErrorAs(t, err, &apiErr)
	require.Same(t, apiErr, err)
	require.Equal(t, "channel_empty", *apiErr.ErrorCode)
	require.Equal(t, "No validated version", *apiErr.ErrorMessage)
	require.Equal(t, 3, calls)
	require.NotContains(t, output, "No published version.")
}

func TestShowConnectorEscapesNameAndCatalogueVersionAsPathSegments(t *testing.T) {
	paths := []string{}
	client := sdkWithTransport(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.EscapedPath())
		require.Empty(t, request.URL.Query().Get("c"))
		body := `{"name":"a/b?c#d","latestVersion":"v/1?x%#"}`
		switch len(paths) {
		case 2:
			body = `{"cursor":{"data":[{"version":"v/1?x%#"}],"hasMore":false,"pageSize":100}}`
		case 3:
			body = `{"version":"v/1?x%#"}`
		}
		return connectorResponse(200, body), nil
	})
	_, err := executeCommand(testNewShowCommand(client), "a/b?c#d")
	require.NoError(t, err)
	require.Equal(t, []string{"/api/connectivity/connectors/a%2Fb%3Fc%23d", "/api/connectivity/connectors/a%2Fb%3Fc%23d/versions", "/api/connectivity/connectors/a%2Fb%3Fc%23d/versions/v%2F1%3Fx%25%23"}, paths)
}

func TestConnectorSDKErrorPreservesResponseBody(t *testing.T) {
	client := sdkWithTransport(func(*http.Request) (*http.Response, error) {
		response := connectorResponse(http.StatusBadGateway, "upstream diagnostic")
		response.Header.Set("Content-Type", "text/plain")
		return response, nil
	})
	_, err := executeCommand(testNewListCommand(client))
	require.ErrorContains(t, err, "upstream diagnostic")
}

func TestConnectorCommandsValidateGeneratedSuccessPayloads(t *testing.T) {
	for _, test := range []struct {
		name        string
		constructor func(*formance.FormanceV1) *cobra.Command
		bodies      []string
		emptyBody   string
		wantError   string
	}{
		{"list", testNewListCommand, []string{`{}`, `{"cursor":null}`, `{"cursor":{}}`, `{"cursor":{"data":null}}`}, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`, "cursor.data must be an array"},
		{"show", testNewShowCommand, []string{`{}`, `{"cursor":null}`, `{"cursor":{}}`, `{"cursor":{"data":null}}`}, `{"cursor":{"data":[],"hasMore":false,"pageSize":100}}`, "cursor.data must be an array"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, body := range append(test.bodies, test.emptyBody) {
				t.Run(body, func(t *testing.T) {
					client := sdkWithTransport(func(request *http.Request) (*http.Response, error) {
						if test.name == "show" && request.URL.Path == "/api/connectivity/connectors/stripe" {
							return connectorResponse(200, `{"name":"stripe"}`), nil
						}
						return connectorResponse(200, body), nil
					})
					command := test.constructor(client)
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

func TestShowConnectorWithoutLatestVersionDoesNotInventAlias(t *testing.T) {
	for _, metadata := range []string{``, `,"latestVersion":""`} {
		t.Run(metadata, func(t *testing.T) {
			calls := 0
			client := sdkWithTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				switch request.URL.Path {
				case "/api/connectivity/connectors/stripe":
					return connectorResponse(200, `{"name":"stripe"`+metadata+`}`), nil
				case "/api/connectivity/connectors/stripe/versions":
					return connectorResponse(200, `{"cursor":{"data":[{"version":"1.0.0"}],"hasMore":false,"pageSize":100}}`), nil
				default:
					t.Fatalf("unexpected version request %s", request.URL.Path)
					return nil, nil
				}
			})
			output, err := executeCommand(testNewShowCommand(client), "stripe")
			require.NoError(t, err)
			require.Equal(t, 2, calls)
			require.Regexp(t, `Available Versions[^\n]*\b1\b`, output)
			require.Contains(t, output, "No latest version schema available.")
			require.NotContains(t, output, "No published version.")
		})
	}
}
