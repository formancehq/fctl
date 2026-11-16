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

func TestV3CatalogUsesEachRecipeChartAndScripts(t *testing.T) {
	for _, tc := range []struct {
		name, pattern string
		templates     int
	}{
		{"omnibus", "banks:{bank_id}:main", 5},
		{"bnpl-lending", "borrowers:{borrower_id}:plans:{plan_id}:installments:{seq}:principal:outstanding", 10},
		{"crypto-custody", "customers:{customer_id}:cash:available", 16},
		{"issuing-financial-host", "cardholder:{account_id}:hold:{authorization_id}", 14},
		{"marketplace-payouts", "sellers:{seller_id}:payable", 10},
		{"neobank-fbo", "customers:{customer_id}:available", 18},
		{"payment-acceptance", "acquirers:{acquirer_id}:main", 4},
		{"rideshare-tutorial", "rider:{rider_id}:ride:{ride_id}:payment", 4},
		{"stablecoin-issuer", "holders:{holder_id}", 13},
		{"stablecoin-operations", "clients:{client_id}:stablecoin", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := loadRecipeSchema(tc.name)
			require.NoError(t, err)
			require.Len(t, schema.V2TransactionTemplates, tc.templates)
			saved := make(map[string]bool)
			sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				response := ""
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/ledger/v3/demo-test":
					require.Equal(t, tc.name, body["metadata"].(map[string]any)["demo.type"])
					require.Equal(t, "STRICT", body["defaultEnforcementMode"])
					var patterns []string
					for _, accountType := range body["accountTypes"].(map[string]any) {
						patterns = append(patterns, accountType.(map[string]any)["pattern"].(string))
					}
					require.Contains(t, patterns, tc.pattern)
					response = `{"data":{"name":"demo-test","mode":"NORMAL","defaultEnforcementMode":"STRICT"}}`
				case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/ledger/v3/demo-test/numscripts/"):
					name := strings.TrimPrefix(r.URL.Path, "/api/ledger/v3/demo-test/numscripts/")
					template, ok := schema.V2TransactionTemplates[name]
					require.True(t, ok)
					require.False(t, saved[name])
					saved[name] = true
					require.Equal(t, template.Script, body["content"])
					require.Equal(t, v3ScriptVersion, body["version"])
					require.Equal(t, "demo-test:"+name+":"+v3ScriptVersion, r.Header.Get("Idempotency-Key"))
					response = `{"data":{"name":"` + name + `","version":"1.0.0"}}`
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				return &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
			})}))
			client := &v3LedgerClient{client: sdk.Ledger.V3, schema: schema}
			_, err = client.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: "demo-test", V2CreateLedgerRequest: ledger.V2CreateLedgerRequest{V2Metadata: map[string]string{"demo.type": tc.name}}})
			require.NoError(t, err)
			_, err = client.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: "demo-test", Version: schemaVersion, V2SchemaData: schema})
			require.NoError(t, err)
			require.Len(t, saved, tc.templates)
		})
	}
}

func TestV3ImplicitChartLeavesAndParentAccounts(t *testing.T) {
	schema, err := loadRecipeSchema("crypto-custody")
	require.NoError(t, err)
	var patterns []string
	for _, accountType := range v3AccountTypes(schema.V2ChartOfAccounts) {
		patterns = append(patterns, accountType.Pattern)
	}
	require.Contains(t, patterns, "customers:{customer_id}")
	require.Contains(t, patterns, "customers:{customer_id}:cash:pending")
	require.Contains(t, patterns, "platform:custody:hot:{network}")
	require.Contains(t, patterns, "external:{network}:{address}")
	require.NotContains(t, patterns, "customers")
	require.NotContains(t, patterns, "customers:{customer_id}:cash")
	bnpl, err := loadRecipeSchema("bnpl-lending")
	require.NoError(t, err)
	types := v3AccountTypes(bnpl.V2ChartOfAccounts)
	require.Equal(t, "^[0-9]+$", *types["borrowers_borrower_id_plans_plan_id_installments_seq_principal_outstanding"].SegmentTypes["seq"].Regex)
}

func TestV3CatalogAccountMetadataDefaults(t *testing.T) {
	for _, tc := range []struct {
		recipe, template string
		vars             map[string]string
		want             map[string]string
	}{
		{"payment-acceptance", "CARD_AUTHORIZATION_GROSS_TOPUP", map[string]string{"acquirer_id": "acquirer", "client_id": "customer", "platform_name": "demo"}, map[string]string{"acquirers:acquirer:main": "acquirer", "clients:customer:main": "wallet"}},
		{"issuing-financial-host", "CARD_AUTHORIZATION_APPROVED", map[string]string{"account_id": "customer", "authorization_id": "auth"}, map[string]string{"cardholder:customer:main": "cardholder", "cardholder:customer:hold:auth": "authorization_hold"}},
		{"stablecoin-operations", "ONRAMP_STEP1_PAYMENT_AUTH_CREDIT", map[string]string{"psp_id": "provider", "client_id": "customer"}, map[string]string{"psp:provider:main": "payment_provider", "clients:customer:stablecoin": "user_stablecoin_balance", "platform:pivot:stablecoin_issuance": "conversion_pivot"}},
	} {
		t.Run(tc.recipe, func(t *testing.T) {
			schema, err := loadRecipeSchema(tc.recipe)
			require.NoError(t, err)
			metadata := v3RecipeAccountMetadata(schema, schema.V2TransactionTemplates[tc.template].Script, tc.vars)
			require.Len(t, metadata, len(tc.want))
			for account, kind := range tc.want {
				require.Equal(t, kind, *metadata[account]["type"].Str)
			}
		})
	}
}

func TestV3CatalogSupportsInlineBootstrapAndExplicitMetadata(t *testing.T) {
	plain := "send [USD/2 1000] (source = @world destination = @cardholder:customer:main)"
	calls := 0
	sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/api/ledger/v3/demo-test/transactions", r.URL.Path)
		require.Equal(t, "run-bootstrap", r.Header.Get("Idempotency-Key"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotContains(t, body, "scriptReference")
		require.Equal(t, plain, body["script"].(map[string]any)["plain"])
		require.Equal(t, "run-bootstrap", body["reference"])
		metadata := body["accountMetadata"].(map[string]any)["cardholder:customer:main"].(map[string]any)
		require.Equal(t, "explicit", metadata["type"])
		require.Equal(t, "bootstrap", metadata["demo.origin"])
		return &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"data":{"transaction":{"id":0,"metadata":{},"postings":[],"reverted":false}}}`)), Request: r}, nil
	})}))
	schema, err := loadRecipeSchema("issuing-financial-host")
	require.NoError(t, err)
	client := &v3LedgerClient{client: sdk.Ledger.V3, schema: schema}
	response, err := client.CreateTransaction(t.Context(), operations.V2CreateTransactionRequest{Ledger: "demo-test", IdempotencyKey: pointer.For("run-bootstrap"), V2PostTransaction: ledger.V2PostTransaction{Reference: pointer.For("run-bootstrap"), V2Metadata: map[string]string{"demo.type": "issuing-financial-host", "demo.template": "DEMO_BOOTSTRAP"}, Script: &ledger.V2PostTransactionScript{Plain: &plain}, AccountMetadata: map[string]map[string]string{"cardholder:customer:main": {"type": "explicit", "demo.origin": "bootstrap"}}}})
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, 1, calls)
}

func TestV3CatalogRejectsAmbiguousScriptSource(t *testing.T) {
	client := &v3LedgerClient{}
	for _, script := range []*ledger.V2PostTransactionScript{nil, {}, {Plain: pointer.For("plain"), Template: pointer.For("template")}} {
		_, err := client.CreateTransaction(t.Context(), operations.V2CreateTransactionRequest{V2PostTransaction: ledger.V2PostTransaction{Script: script}})
		require.ErrorContains(t, err, "exactly one")
	}
}

func testRecipeSchema(t *testing.T, name string) ledger.V2SchemaDataInput {
	t.Helper()
	schema, err := loadRecipeSchema(name)
	require.NoError(t, err)
	return schema
}

func TestV3ClientUsesInjectedSchemaWithoutRecipeMetadata(t *testing.T) {
	calls := 0
	sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body["accountTypes"].(map[string]any), "acquirers_acquirer_id_main")
		require.NotContains(t, body["accountTypes"].(map[string]any), "banks_bank_id_payout_payout_ref")
		return &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"data":{"name":"demo-test","mode":"NORMAL","defaultEnforcementMode":"STRICT"}}`)), Request: r}, nil
	})}))
	client := &v3LedgerClient{client: sdk.Ledger.V3, schema: testRecipeSchema(t, "payment-acceptance")}
	_, err := client.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: "demo-test"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}
