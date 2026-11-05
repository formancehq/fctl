package schemas

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const schemaFixture = `{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{},"queries":{"accounts":{"resource":"accounts","params":{"pageSize":15,"sort":"address:asc"}}}}`

type schemaRoundTripper func(*http.Request) (*http.Response, error)

func (f schemaRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func schemaClients(f schemaRoundTripper) *fctl.StackClients {
	return &fctl.StackClients{URI: "https://stack.example.test", HTTPClient: &http.Client{Transport: f}}
}

func schemaResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestGetSchemaPreservesParamsWithoutNestedResource(t *testing.T) {
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/ledger/v2/default/schemas/v1", r.URL.Path)
		return schemaResponse(http.StatusOK, `{"data":`+schemaFixture+`}`), nil
	})
	schema, err := getSchema(context.Background(), clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.NoError(t, err)
	data, err := json.Marshal(schema)
	require.NoError(t, err)
	require.JSONEq(t, schemaFixture, string(data))
}

func TestListSchemasPreservesParamsAndPagination(t *testing.T) {
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/api/ledger/v2/default/schemas", r.URL.Path)
		require.Equal(t, "opaque-token", r.URL.Query().Get("cursor"))
		return schemaResponse(http.StatusOK, `{"cursor":{"data":[`+schemaFixture+`],"hasMore":true,"pageSize":15,"next":"next-token"}}`), nil
	})
	cursor := "opaque-token"
	page, err := listSchemas(context.Background(), clients, operations.V2ListSchemasRequest{Ledger: "default", Cursor: &cursor})
	require.NoError(t, err)
	require.True(t, page.HasMore)
	require.Equal(t, int64(15), page.PageSize)
	require.Equal(t, "next-token", *page.Next)
	require.Len(t, page.Data, 1)
	data, err := json.Marshal(page.Data[0])
	require.NoError(t, err)
	require.JSONEq(t, schemaFixture, string(data))
}

func TestInsertSchemaPreservesParamsWithoutNestedResource(t *testing.T) {
	const body = `{"chart":{},"queries":{"accounts":{"resource":"accounts","params":{"pageSize":15,"sort":"address:asc"}},"volumes":{"resource":"volumes","params":{"groupBy":2,"insertionDate":false}}}}`
	var data SchemaData
	require.NoError(t, json.Unmarshal([]byte(body), &data))
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/ledger/v2/default/schemas/v1", r.URL.Path)
		actual, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, body, string(actual))
		require.Equal(t, int64(len(actual)), r.ContentLength)
		return schemaResponse(http.StatusNoContent, ""), nil
	})
	require.NoError(t, insertSchema(context.Background(), clients, operations.V2InsertSchemaRequest{Ledger: "default", Version: "v1"}, &data))
}

func TestSchemaTransportPreservesCancellationAndHTTPError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clients := schemaClients(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	_, err := getSchema(ctx, clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.True(t, errors.Is(err, context.Canceled))

	clients = schemaClients(func(r *http.Request) (*http.Response, error) {
		return schemaResponse(http.StatusBadRequest, `{"errorCode":"VALIDATION","errorMessage":"invalid schema"}`), nil
	})
	_, err = getSchema(context.Background(), clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.ErrorContains(t, err, "invalid schema")
}

func TestLoadSchemaYAMLWithoutNestedResource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.yaml")
	require.NoError(t, os.WriteFile(path, []byte("chart: {}\nqueries:\n  accounts:\n    resource: accounts\n    params:\n      pageSize: 15\n"), 0600))
	data, err := loadSchemaData(&cobra.Command{}, path)
	require.NoError(t, err)
	encoded, err := json.Marshal(data)
	require.NoError(t, err)
	require.JSONEq(t, `{"chart":{},"queries":{"accounts":{"resource":"accounts","params":{"pageSize":15}}}}`, string(encoded))
}

func TestSchemaRequestsUseConfiguredServerAndTransport(t *testing.T) {
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "https://stack.example.test/proxy/api/ledger/v2/default/schemas/v1", r.URL.String())
		return schemaResponse(http.StatusOK, `{"data":`+schemaFixture+`}`), nil
	})
	clients.URI += "/proxy"
	_, err := getSchema(context.Background(), clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.NoError(t, err)
}

func TestGetSchemaRejectsUnexpectedContentType(t *testing.T) {
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		response := schemaResponse(http.StatusOK, `{"data":`+schemaFixture+`}`)
		response.Header.Set("Content-Type", "text/plain")
		return response, nil
	})
	_, err := getSchema(context.Background(), clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.Error(t, err)
}

func TestSchemaAdapterPreservesLegacyTimeBounds(t *testing.T) {
	const input = `{"chart":{},"queries":{"accounts":{"resource":"accounts","params":{"resource":"accounts","pageSize":15,"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}}}}`
	const fixture = `{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{},"queries":{"accounts":{"resource":"accounts","params":{"resource":"accounts","pageSize":15,"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}}}}`
	clients := schemaClients(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/api/ledger/v2/default/schemas/v1", r.URL.Path)
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.JSONEq(t, input, string(body))
			return schemaResponse(http.StatusNoContent, ""), nil
		}
		require.Equal(t, http.MethodGet, r.Method)
		return schemaResponse(http.StatusOK, `{"data":`+fixture+`}`), nil
	})
	var data SchemaData
	require.NoError(t, json.Unmarshal([]byte(input), &data))
	require.NoError(t, insertSchema(t.Context(), clients, operations.V2InsertSchemaRequest{Ledger: "default", Version: "v1"}, &data))
	got, err := getSchema(t.Context(), clients, operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
	require.NoError(t, err)
	actual, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, fixture, string(actual))
}
