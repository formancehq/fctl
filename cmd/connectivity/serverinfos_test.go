package connectivity

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	connectivitymodel "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type statusContextKey struct{}

func executeServerInfos(ctx context.Context, client *formance.FormanceV1, args ...string) (string, error) {
	command := NewServerInfosCommand()
	controller := &ServerInfosController{store: &connectivitymodel.Info{}}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		renderable, err := controller.run(cmd, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, renderable)
	}
	command.Flags().String(fctl.OutputFlag, "plain", "")
	command.SilenceErrors = true
	command.SilenceUsage = true
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	err := command.ExecuteContext(ctx)
	return out.String(), err
}

func TestServerInfosShowsBuildInformationInPlainAndFlatJSON(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		t.Run(format, func(t *testing.T) {
			calls := 0
			ctx := context.WithValue(t.Context(), statusContextKey{}, "status caller")
			sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: statusRoundTripper(func(request *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "status caller", request.Context().Value(statusContextKey{}))
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}`))}, nil
			})}))
			out, err := executeServerInfos(ctx, sdk.Connectivity.V1, "--output", format)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			if format == "json" {
				require.JSONEq(t, `{"data":{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}}`, out)
			} else {
				require.Equal(t, "Version    | v1.2.3              \nCommit     | abc123              \nBuild Date | 2026-10-28T22:29:14Z\n\n", out)
			}
		})
	}
}

func TestServerInfosReturnsTransportAndEmptyResponseErrors(t *testing.T) {
	transportErr := errors.New("upstream unavailable")
	for _, test := range []struct {
		name         string
		body         string
		transportErr error
		want         string
	}{
		{"transport", "", transportErr, "upstream unavailable"},
		{"empty response", "", nil, "unexpected end of JSON input"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, format := range []string{"plain", "json"} {
				sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: statusRoundTripper(func(*http.Request) (*http.Response, error) {
					if test.transportErr != nil {
						return nil, test.transportErr
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(test.body))}, nil
				})}))
				out, err := executeServerInfos(t.Context(), sdk.Connectivity.V1, "--output", format)
				require.ErrorContains(t, err, test.want)
				require.Empty(t, out)
				if test.transportErr != nil {
					require.ErrorIs(t, err, test.transportErr)
				}
			}
		})
	}
}

func TestServerInfosRejectsPositionalArgumentsBeforeAuthentication(t *testing.T) {
	command := NewServerInfosCommand()
	command.SetArgs([]string{"extra"})
	command.SilenceErrors = true
	command.SilenceUsage = true
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(io.Discard)
	require.ErrorContains(t, command.Execute(), "unknown command")
	require.Empty(t, out.String())
}

type statusRoundTripper func(*http.Request) (*http.Response, error)

func (f statusRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestServerInfosUsesGeneratedSDKAndRejectsErrorEnvelope(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized} {
		sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: statusRoundTripper(func(request *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, request.Method)
			require.Equal(t, "/api/connectivity/_info", request.URL.Path)
			body := `{"errorCode":"unavailable","errorMessage":"service unavailable"}`
			if status == http.StatusOK {
				body = `{"version":"v1","commit":"abc","buildDate":"today"}`
			}
			rawResponse := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
			return rawResponse, nil
		})}))
		out, err := executeServerInfos(t.Context(), sdk.Connectivity.V1, "--output", "json")
		if status == http.StatusOK {
			require.NoError(t, err)
			require.JSONEq(t, `{"data":{"version":"v1","commit":"abc","buildDate":"today"}}`, out)
		} else {
			require.Error(t, err)
			var apiErr *connectivitymodel.Error
			require.ErrorAs(t, err, &apiErr)
			require.Same(t, apiErr, err)
			require.Equal(t, "unavailable", *apiErr.ErrorCode)
			require.Equal(t, "service unavailable", *apiErr.ErrorMessage)
			require.Empty(t, out)
		}
	}
}

func TestServerInfosRequiresStringBuildFieldsAndRetainsResponseBody(t *testing.T) {
	for _, test := range []struct {
		body  string
		valid bool
	}{
		{`{}`, false},
		{`null`, false},
		{`{"commit":"","buildDate":""}`, false},
		{`{"version":"","buildDate":""}`, false},
		{`{"version":"","commit":""}`, false},
		{`{"version":null,"commit":"","buildDate":""}`, false},
		{`{"version":"","commit":null,"buildDate":""}`, false},
		{`{"version":"","commit":"","buildDate":null}`, false},
		{`{"version":1,"commit":"","buildDate":""}`, false},
		{`{"version":"","commit":false,"buildDate":""}`, false},
		{`{"version":"","commit":"","buildDate":{}}`, false},
		{`{"version":"","commit":"","buildDate":""}`, true},
		{`{"version":"v1","commit":"abc","buildDate":"today"}`, true},
	} {
		t.Run(test.body, func(t *testing.T) {
			for _, format := range []string{"plain", "json"} {
				var rawResponse *http.Response
				sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: statusRoundTripper(func(*http.Request) (*http.Response, error) {
					rawResponse = &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(test.body))}
					return rawResponse, nil
				})}))
				out, err := executeServerInfos(t.Context(), sdk.Connectivity.V1, "--output", format)
				if test.valid {
					require.NoError(t, err)
					require.NotEmpty(t, out)
				} else {
					require.Error(t, err)
					require.Empty(t, out)
				}
				remaining, readErr := io.ReadAll(rawResponse.Body)
				require.NoError(t, readErr)
				require.Equal(t, test.body, string(remaining))
			}
		})
	}
}
