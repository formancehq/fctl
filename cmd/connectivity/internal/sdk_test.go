package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
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

func TestSDKResponseError(t *testing.T) {
	require.NoError(t, ResponseError(http.StatusNoContent, nil, nil))
	err := ResponseError(http.StatusUnauthorized, nil, nil)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Equal(t, "Unauthorized", apiErr.Message)
	require.Empty(t, apiErr.Body)
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
	require.NoError(t, ResponseError(resp.StatusCode, resp.Error, resp.RawResponse))
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
	require.NoError(t, err)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, ResponseError(resp.StatusCode, resp.Error, resp.RawResponse), &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, `{"errorCode":"invalid_query","errorMessage":"bad query","details":"{\"field\":\"query\"}","unknown":9007199254740993}`, apiErr.Body)
	require.Same(t, resp.RawResponse, apiErr.RawResponse)
	raw, err := io.ReadAll(resp.RawResponse.Body)
	require.NoError(t, err)
	require.Equal(t, apiErr.Body, string(raw))
}

func TestSDKResponseErrorSerializesCurrentModelWithoutRawResponse(t *testing.T) {
	code, message, details := "invalid_query", "bad query", `{"field":"query"}`
	err := ResponseError(http.StatusBadRequest, &connectivity.Error{ErrorCode: &code, ErrorMessage: &message, Details: &details}, nil)
	var apiErr *sdkerrors.SDKError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.JSONEq(t, `{"errorCode":"invalid_query","errorMessage":"bad query","details":"{\"field\":\"query\"}"}`, apiErr.Body)
	require.Nil(t, apiErr.RawResponse)
}
