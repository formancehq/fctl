package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

func TestSDKPatchPreservesExplicitMergePatchBody(t *testing.T) {
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/proxy/api/connectivity/connections/stripe", r.URL.Path)
		require.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
		var err error
		received, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"stripe","configuration":{}}`)
	}))
	defer server.Close()
	body := []byte(`{"configuration":{"channel":null,"env":{"KEY":null},"files":{}}}`)
	ctx := WithPatchBody(t.Context(), body)
	response, err := PatchAPI(&fctl.StackClients{URI: server.URL + "/proxy", HTTPClient: server.Client()}).PatchConnection(ctx, operations.PatchConnectionRequest{Name: "stripe"}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.JSONEq(t, string(body), string(received))
}

func TestSDKListConnectionsTransmitsQueryAndCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/connectivity/connections", r.URL.Path)
		require.Equal(t, "next page", r.URL.Query().Get("cursor"))
		require.Equal(t, "2", r.URL.Query().Get("pageSize"))
		require.Equal(t, `{"$match":{"connector":"stripe"}}`, r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"cursor":{"pageSize":2,"hasMore":true,"next":"next-token","data":[{"name":"stripe","configuration":{}}]}}`)
	}))
	defer server.Close()
	size := 2
	resp, err := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client())).Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{
		Cursor: OptionalString("next page"), PageSize: &size, Query: OptionalString(`{"$match":{"connector":"stripe"}}`),
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, resp.ConnectionCursor)
	require.True(t, resp.ConnectionCursor.Cursor.HasMore)
	require.Equal(t, "next-token", *resp.ConnectionCursor.Cursor.Next)
	require.Equal(t, "stripe", resp.ConnectionCursor.Cursor.Data[0].Name)
}

func TestSDKStructuredErrorUsesResponseStatusAndCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"errorCode":"invalid_query","errorMessage":"bad query","details":"{\"field\":\"query\"}","unknown":9007199254740993}`)
	}))
	defer server.Close()
	resp, err := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client())).Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
	require.Error(t, err)
	require.Nil(t, resp)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, `{"errorCode":"invalid_query","errorMessage":"bad query","details":"{\"field\":\"query\"}","unknown":9007199254740993}`, apiErr.Body)
	require.NotNil(t, apiErr.RawResponse)
	raw, err := io.ReadAll(apiErr.RawResponse.Body)
	require.NoError(t, err)
	require.Equal(t, apiErr.Body, string(raw))
}

func TestSDKFailureClassesReturnOnlySDKError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			body := `{"errorCode":"probe_error","errorMessage":"probe message","details":"probe details","unknown":9007199254740993}`
			if status == http.StatusUnauthorized {
				body = ""
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			client := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client()))
			response, err := client.Connectivity.V1.ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
			require.Error(t, err)
			require.Nil(t, response)
			var apiError *sdkerrors.SDKError
			require.ErrorAs(t, err, &apiError)
			require.Equal(t, status, apiError.StatusCode)
			require.Equal(t, body, apiError.Body)
			require.NotNil(t, apiError.RawResponse)
			require.Equal(t, status, apiError.RawResponse.StatusCode)
			raw, readErr := io.ReadAll(apiError.RawResponse.Body)
			require.NoError(t, readErr)
			require.Equal(t, body, string(raw))
		})
	}
}

func TestSDKHealthFailureReturnsOnlySDKError(t *testing.T) {
	const body = `{"storage":"database unavailable"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/connectivity/_healthcheck", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	client := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client()))
	response, err := client.Connectivity.V1.Healthcheck(t.Context())
	require.Error(t, err)
	require.Nil(t, response)
	var apiError *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusInternalServerError, apiError.StatusCode)
	require.Equal(t, body, apiError.Body)
	require.NotNil(t, apiError.RawResponse)
	require.Equal(t, http.StatusInternalServerError, apiError.RawResponse.StatusCode)
	raw, readErr := io.ReadAll(apiError.RawResponse.Body)
	require.NoError(t, readErr)
	require.Equal(t, body, string(raw))
}
