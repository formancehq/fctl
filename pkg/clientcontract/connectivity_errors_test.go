package clientcontract

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/retry"
)

type connectivityErrorTransport func(*http.Request) (*http.Response, error)

func (f connectivityErrorTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }

func connectivityErrorSDK(t *testing.T, status int, contentType, body string) *formance.Formance {
	t.Helper()
	return formance.New(
		formance.WithServerURL("https://stack.example.test"),
		formance.WithRetryConfig(retry.Config{Strategy: "none"}),
		formance.WithClient(connectivityErrorTransport(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/api/connectivity/connections", r.URL.Path)
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"Content-Type": {contentType}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		})),
	)
}

func TestConnectivityApplicationFailuresUseErrorChannel(t *testing.T) {
	const details = `{"field":"query","number":9007199254740993}`
	body, err := json.Marshal(map[string]string{"errorCode": "example_failure", "errorMessage": "Request failed", "details": details})
	require.NoError(t, err)
	for _, status := range []int{400, 403, 404, 409, 429, 499, 500, 503, 599} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			sdk := connectivityErrorSDK(t, status, "application/json", string(body))
			response, err := sdk.Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
			require.Nil(t, response)
			var apiErr *connectivity.Error
			require.ErrorAs(t, err, &apiErr)
			require.NotNil(t, apiErr.ErrorCode)
			require.NotNil(t, apiErr.ErrorMessage)
			require.NotNil(t, apiErr.Details)
			require.Equal(t, "example_failure", *apiErr.ErrorCode)
			require.Equal(t, "Request failed", *apiErr.ErrorMessage)
			require.Equal(t, details, *apiErr.Details)
			require.JSONEq(t, string(body), apiErr.Error())
		})
	}
}

func TestConnectivityBodylessUnauthorizedPreservesHTTPStatus(t *testing.T) {
	sdk := connectivityErrorSDK(t, http.StatusUnauthorized, "", "")
	response, err := sdk.Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
	require.Nil(t, response)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Empty(t, apiErr.Body)
	require.NotNil(t, apiErr.RawResponse)
	require.Equal(t, http.StatusUnauthorized, apiErr.RawResponse.StatusCode)
}

func TestConnectivityNonJSONFailurePreservesHTTPResponse(t *testing.T) {
	const body = "temporarily unavailable\n"
	sdk := connectivityErrorSDK(t, http.StatusServiceUnavailable, "text/plain", body)
	response, err := sdk.Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
	require.Nil(t, response)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
	require.Equal(t, body, apiErr.Body)
	require.NotNil(t, apiErr.RawResponse)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.RawResponse.StatusCode)
}
