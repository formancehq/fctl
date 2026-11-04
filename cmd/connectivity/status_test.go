package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	connectivitymodel "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type statusClientMock struct {
	connectivityinternal.API
	getInfo func(context.Context) (*connectivitymodel.Info, error)
}

func (m statusClientMock) ConnectivityInfo(ctx context.Context, _ ...operations.Option) (*operations.ConnectivityInfoResponse, error) {
	info, err := m.getInfo(ctx)
	body, marshalErr := json.Marshal(info)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return &operations.ConnectivityInfoResponse{StatusCode: 200, Info: info, RawResponse: &http.Response{Body: io.NopCloser(bytes.NewReader(body))}}, err
}

type statusContextKey struct{}

func executeStatus(ctx context.Context, factory connectivityinternal.ClientFactory, args ...string) (string, error) {
	command := NewStatusCommand(factory)
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

func TestStatusShowsBuildInformationInPlainAndFlatJSON(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		t.Run(format, func(t *testing.T) {
			calls := 0
			ctx := context.WithValue(t.Context(), statusContextKey{}, "status caller")
			factory := func(cmd *cobra.Command) (connectivityinternal.API, error) {
				require.Same(t, ctx, cmd.Context())
				return statusClientMock{getInfo: func(got context.Context) (*connectivitymodel.Info, error) {
					calls++
					require.Same(t, ctx, got)
					return &connectivitymodel.Info{Version: "v1.2.3", Commit: "abc123", BuildDate: "2026-10-28T22:29:14Z"}, nil
				}}, nil
			}
			out, err := executeStatus(ctx, factory, "--output", format)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			if format == "json" {
				require.JSONEq(t, `{"data":{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}}`, out)
			} else {
				require.Equal(t, "Version: v1.2.3\nCommit: abc123\nBuild Date: 2026-10-28T22:29:14Z\n", out)
			}
		})
	}
}

func TestStatusReturnsFactoryAPINilClientAndEmptyResponseErrors(t *testing.T) {
	apiErr := &connectivityinternal.APIError{StatusCode: 401, Message: "Unauthorized"}
	for _, tt := range []struct {
		name    string
		factory connectivityinternal.ClientFactory
		want    string
	}{
		{"missing factory", nil, "factory is required"},
		{"factory", func(*cobra.Command) (connectivityinternal.API, error) {
			return nil, errors.New("authentication failed")
		}, "authentication failed"},
		{"missing client", func(*cobra.Command) (connectivityinternal.API, error) { return nil, nil }, "client is required"},
		{"API", func(*cobra.Command) (connectivityinternal.API, error) {
			return statusClientMock{getInfo: func(context.Context) (*connectivitymodel.Info, error) { return nil, apiErr }}, nil
		}, "Unauthorized"},
		{"empty response", func(*cobra.Command) (connectivityinternal.API, error) {
			return statusClientMock{getInfo: func(context.Context) (*connectivitymodel.Info, error) { return nil, nil }}, nil
		}, "empty response"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, format := range []string{"plain", "json"} {
				out, err := executeStatus(t.Context(), tt.factory, "--output", format)
				require.ErrorContains(t, err, tt.want)
				require.Empty(t, out)
				if tt.name == "API" {
					require.ErrorIs(t, err, apiErr)
				}
			}
		})
	}
}

func TestStatusRejectsPositionalArgumentsBeforeAuthentication(t *testing.T) {
	called := false
	factory := func(*cobra.Command) (connectivityinternal.API, error) { called = true; return nil, nil }
	out, err := executeStatus(t.Context(), factory, "extra")
	require.Error(t, err)
	require.False(t, called)
	require.Empty(t, out)
}

type statusRoundTripper func(*http.Request) (*http.Response, error)

func (f statusRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestStatusUsesGeneratedSDKAndRejectsErrorEnvelope(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized} {
		sdk := formance.New(formance.WithServerURL("https://stack.example.test"), formance.WithClient(&http.Client{Transport: statusRoundTripper(func(request *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, request.Method)
			require.Equal(t, "/api/connectivity/_info", request.URL.Path)
			body := `{"code":"unavailable","message":"service unavailable"}`
			if status == http.StatusOK {
				body = `{"version":"v1","commit":"abc","buildDate":"today"}`
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}))
		factory := func(*cobra.Command) (connectivityinternal.API, error) { return sdk.Connectivity.V1, nil }
		out, err := executeStatus(t.Context(), factory, "--output", "json")
		if status == http.StatusOK {
			require.NoError(t, err)
			require.JSONEq(t, `{"data":{"version":"v1","commit":"abc","buildDate":"today"}}`, out)
		} else {
			require.Error(t, err)
			var apiErr *connectivityinternal.APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, status, apiErr.StatusCode)
			require.Empty(t, out)
		}
	}
}

func TestStatusRequiresStringBuildFieldsAndRetainsResponseBody(t *testing.T) {
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
				factory := func(*cobra.Command) (connectivityinternal.API, error) { return sdk.Connectivity.V1, nil }
				out, err := executeStatus(t.Context(), factory, "--output", format)
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
