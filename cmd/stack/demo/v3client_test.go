package demo

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/pointer"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func TestV3ClientUsesLedgerV3RoutesAndSavedRecipeScripts(t *testing.T) {
	schema, err := omnibusSchema()
	require.NoError(t, err)
	var saved []string
	var sent int
	sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		response := ""
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/ledger/v3/demo-test":
			require.Equal(t, "demo-test:create", r.Header.Get("Idempotency-Key"))
			require.Equal(t, "STRICT", body["defaultEnforcementMode"])
			require.Equal(t, "NORMAL", body["mode"])
			require.Equal(t, "omnibus", body["metadata"].(map[string]any)["demo.type"])
			types := body["accountTypes"].(map[string]any)
			require.Len(t, types, 6)
			require.Equal(t, "banks:{bank_id}:main", types["banks_bank_id_main"].(map[string]any)["pattern"])
			require.Equal(t, "clients:{client_id}:main", types["clients_client_id_main"].(map[string]any)["pattern"])
			response = `{"data":{"name":"demo-test","mode":"NORMAL","defaultEnforcementMode":"STRICT"}}`
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/ledger/v3/demo-test/numscripts/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/ledger/v3/demo-test/numscripts/")
			require.Equal(t, schema.V2TransactionTemplates[name].Script, body["content"])
			require.Equal(t, "1.0.0", body["version"])
			require.Equal(t, "demo-test:"+name+":1.0.0", r.Header.Get("Idempotency-Key"))
			saved = append(saved, name)
			response = `{"data":{"name":"` + name + `","version":"1.0.0"}}`
		case r.Method == http.MethodPost && r.URL.Path == "/api/ledger/v3/demo-test/transactions":
			require.Equal(t, "run-0", r.Header.Get("Idempotency-Key"))
			require.Equal(t, "run-0", body["reference"])
			require.NotContains(t, body, "script")
			reference := body["scriptReference"].(map[string]any)
			require.Equal(t, "CLIENT_DEPOSIT", reference["name"])
			require.Equal(t, "1.0.0", reference["version"])
			require.Equal(t, "1000", reference["vars"].(map[string]any)["amount"])
			require.Equal(t, "run", body["metadata"].(map[string]any)["demo.run"])
			metadata := body["accountMetadata"].(map[string]any)
			require.Equal(t, "nostro", metadata["banks:demo_bank:main"].(map[string]any)["type"])
			require.Equal(t, "vostro", metadata["clients:client_00:main"].(map[string]any)["type"])
			sent++
			response = `{"data":{"transaction":{"id":0,"metadata":{},"postings":[],"reverted":false}}}`
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
	})}))
	client := &v3LedgerClient{client: sdk.Ledger.V3, schema: testRecipeSchema(t, "omnibus")}
	created, err := client.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: "demo-test", V2CreateLedgerRequest: ledger.V2CreateLedgerRequest{V2Metadata: map[string]string{"demo.type": "omnibus"}}})
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, created.StatusCode)
	inserted, err := client.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: "demo-test", Version: "v1", V2SchemaData: schema})
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, inserted.StatusCode)
	require.ElementsMatch(t, []string{"CLIENT_DEPOSIT", "UNIDENTIFIED_DEPOSIT", "SUSPENSE_RESOLUTION", "PAYOUT_RESERVE", "PAYOUT_SETTLEMENT"}, saved)
	response, err := client.CreateTransaction(t.Context(), operations.V2CreateTransactionRequest{
		Ledger: "demo-test", IdempotencyKey: pointer.For("run-0"), V2PostTransaction: ledger.V2PostTransaction{
			Reference: pointer.For("run-0"), V2Metadata: map[string]string{"demo.run": "run"}, Script: &ledger.V2PostTransactionScript{Template: pointer.For("CLIENT_DEPOSIT"), Vars: map[string]string{"bank_id": "demo_bank", "client_id": "client_00", "amount": "1000", "asset": "EUR/2", "reference": "run-0"}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, 1, sent)
}

func TestV3ClientDoesNotAcknowledgeErrorResponses(t *testing.T) {
	sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"errorCode":"PERMISSION_DENIED","errorMessage":"missing write scope"}`)), Request: r}, nil
	})}))
	client := &v3LedgerClient{client: sdk.Ledger.V3, schema: testRecipeSchema(t, "omnibus")}
	_, err := client.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: "demo-test"})
	var apiErr *v3APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	require.Equal(t, "PERMISSION_DENIED", apiErr.Response.ErrorCode)
	require.ErrorContains(t, err, "missing write scope")
	schema, err := omnibusSchema()
	require.NoError(t, err)
	_, err = client.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: "demo-test", V2SchemaData: schema})
	require.ErrorContains(t, err, "PERMISSION_DENIED")
	_, err = client.CreateTransaction(t.Context(), operations.V2CreateTransactionRequest{Ledger: "demo-test", V2PostTransaction: ledger.V2PostTransaction{Script: &ledger.V2PostTransactionScript{Template: pointer.For("CLIENT_DEPOSIT")}}})
	require.ErrorContains(t, err, "PERMISSION_DENIED")
}
