package clientcontract

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ledgerTransport func(*http.Request) (*http.Response, error)

func (f ledgerTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }
func ledgerResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func ledgerSDK(f ledgerTransport) *formance.Formance {
	return formance.New(formance.WithServerURL("https://stack.example.test/proxy"), formance.WithClient(f))
}
func queryJSON(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &fields))
	return string(fields["queries"])
}

func TestLedgerOriginalInfoContract(t *testing.T) {
	sdk := ledgerSDK(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "GET /proxy/api/ledger/_info", r.Method+" "+r.URL.Path)
		return ledgerResponse(200, `{"data":{"server":"ledger","version":"v2.4.18","config":{"storage":{"driver":"postgres"}}}}`), nil
	})
	response, err := sdk.Ledger.V1.GetInfo(t.Context())
	require.NoError(t, err)
	require.Equal(t, "ledger", response.ConfigInfoResponse.ConfigInfo.Server)
	require.Equal(t, "v2.4.18", response.ConfigInfoResponse.ConfigInfo.Version)
}

func TestLedgerPlainQueryParamsThroughSDK(t *testing.T) {
	for _, resource := range []string{"accounts", "transactions", "logs", "volumes"} {
		t.Run(resource, func(t *testing.T) {
			params := `{"pageSize":15,"sort":"id:asc","expand":["volumes"],"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}`
			if resource == "volumes" {
				params = `{"pageSize":15,"groupBy":2,"insertionDate":false,"expand":["volumes"],"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}`
			}
			queries := `{"query":{"resource":"` + resource + `","params":` + params + `}}`
			input := `{"chart":{},"queries":` + queries + `}`
			fixture := `{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{},"queries":` + queries + `}`
			sdk := ledgerSDK(func(r *http.Request) (*http.Response, error) {
				require.Contains(t, []string{"GET /proxy/api/ledger/v2/default/schemas/v1", "POST /proxy/api/ledger/v2/default/schemas/v1", "GET /proxy/api/ledger/v2/default/schemas"}, r.Method+" "+r.URL.Path)
				switch {
				case r.Method == http.MethodPost:
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.JSONEq(t, input, string(body))
					return ledgerResponse(204, ""), nil
				case strings.HasSuffix(r.URL.Path, "/v1"):
					return ledgerResponse(200, `{"data":`+fixture+`}`), nil
				default:
					require.Equal(t, "opaque-token", r.URL.Query().Get("cursor"))
					return ledgerResponse(200, `{"cursor":{"data":[`+fixture+`],"hasMore":true,"pageSize":15,"next":"next-token"}}`), nil
				}
			})
			var data ledger.V2SchemaDataInput
			require.NoError(t, json.Unmarshal([]byte(input), &data))
			require.JSONEq(t, queries, queryJSON(t, data))
			inserted, err := sdk.Ledger.V2.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: "default", Version: "v1", V2SchemaData: data})
			require.NoError(t, err)
			require.Equal(t, 204, inserted.StatusCode)
			got, err := sdk.Ledger.V2.GetSchema(t.Context(), operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"})
			require.NoError(t, err)
			require.JSONEq(t, queries, queryJSON(t, got.V2SchemaResponse.V2SchemaData))
			cursor := "opaque-token"
			listed, err := sdk.Ledger.V2.ListSchemas(t.Context(), operations.V2ListSchemasRequest{Ledger: "default", Cursor: &cursor})
			require.NoError(t, err)
			page := listed.V2SchemasCursorResponse.V2SchemasCursor
			require.True(t, page.HasMore)
			require.Equal(t, int64(15), page.PageSize)
			require.Equal(t, "next-token", *page.Next)
			require.Len(t, page.Data, 1)
			require.JSONEq(t, queries, queryJSON(t, page.Data[0]))
		})
	}
}

func TestLedgerSchemaLogDecodesWithoutFallback(t *testing.T) {
	sdk := ledgerSDK(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/proxy/api/ledger/v2/default/logs", r.URL.Path)
		require.Equal(t, "1", r.URL.Query().Get("pageSize"))
		return ledgerResponse(200, `{"cursor":{"pageSize":1,"hasMore":false,"data":[{"id":9007199254740993,"type":"INSERTED_SCHEMA","hash":"hash","date":"2026-01-01T00:00:00Z","data":{"schema":{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{},"queries":{"accounts":{"resource":"accounts","params":{"pageSize":10,"expand":["volumes"],"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}}}}}}]}}`), nil
	})
	pageSize := int64(1)
	got, err := sdk.Ledger.V2.ListLogs(t.Context(), operations.V2ListLogsRequest{Ledger: "default", PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, got.V2LogsCursorResponse.Cursor.Data, 1)
	require.Equal(t, "9007199254740993", got.V2LogsCursorResponse.Cursor.Data[0].ID.String())
}

func TestLedgerSchemaErrorsAndCancellation(t *testing.T) {
	sdk := ledgerSDK(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		return ledgerResponse(400, `{"errorCode":"VALIDATION","errorMessage":"invalid schema"}`), nil
	})
	request := operations.V2GetSchemaRequest{Ledger: "default", Version: "v1"}
	_, err := sdk.Ledger.V2.GetSchema(t.Context(), request)
	require.ErrorContains(t, err, "invalid schema")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = sdk.Ledger.V2.GetSchema(ctx, request)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestLedgerV3InfoContractRemainsSeparate(t *testing.T) {
	sdk := ledgerSDK(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "GET /proxy/api/ledger/_info", r.Method+" "+r.URL.Path)
		require.Empty(t, r.URL.Fragment)
		return ledgerResponse(200, `{"version":"v3.0.0","commit":"abcdef","buildDate":"2026-01-01"}`), nil
	})
	response, err := sdk.Ledger.V3.Info(t.Context(), &operations.V3InfoSecurity{})
	require.NoError(t, err)
	require.Equal(t, "v3.0.0", response.ServerInfo.Version)
}
