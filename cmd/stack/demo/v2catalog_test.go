package demo

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

// Exercise the real generated Ledger v2 SDK for the entire recipe catalog.
// Responses are transport fixtures; this is wire-contract coverage, not proof
// that these requests were accepted by a deployed Ledger v2 server.
func TestV2CatalogPreservesSchemasAndRecipeRequestsOnWire(t *testing.T) {
	totalTemplates, totalQueries := 0, 0
	for _, name := range recipeNames() {
		t.Run(name, func(t *testing.T) {
			schema, err := loadRecipeSchema(name)
			require.NoError(t, err)
			original := v2CatalogSourceSchema(t, name)
			sourceTemplates := original["transactions"].(map[string]any)
			totalTemplates += len(sourceTemplates)
			sourceQueries, _ := original["queries"].(map[string]any)
			totalQueries += len(sourceQueries)
			// The docs may supply a display name; the API identifies a query by
			// its map key and has no name member in its query-template contract.
			for _, value := range sourceQueries {
				delete(value.(map[string]any), "name")
			}
			// Runtime is an execution normalization; the original script bodies stay
			// byte-for-byte intact. Rideshare's documentation hints are not API fields.
			for _, value := range sourceTemplates {
				template := value.(map[string]any)
				template["runtime"] = "experimental-interpreter"
				delete(template, "interpreter")
				delete(template, "featureFlags")
			}
			sourceChart := original["chart"].(map[string]any)
			if _, exists := sourceChart["world"]; !exists {
				if _, effective := schema.V2ChartOfAccounts["world"]; effective {
					sourceChart["world"] = map[string]any{".self": map[string]any{}}
				}
			}
			created, installed, sent := 0, 0, 0
			var expected operations.V2CreateTransactionRequest
			seen := map[string]bool{}
			sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				status, response := http.StatusNoContent, ""
				switch r.URL.Path {
				case "/api/ledger/v2/demo-test":
					created++
					require.Equal(t, map[string]any{"demo.type": name, "demo.run": "wire-run"}, body["metadata"])
				case "/api/ledger/v2/demo-test/schemas/v1":
					installed++
					require.Equal(t, 1, created)
					require.Equal(t, sourceChart, body["chart"], "implicit chart leaves and metadata must survive serialization")
					require.Equal(t, sourceTemplates, body["transactions"], "all official script bodies and runtimes must survive serialization")
					if len(sourceQueries) > 0 {
						require.Equal(t, sourceQueries, body["queries"], "query discriminators, variables, body and params must survive serialization")
					} else {
						require.NotContains(t, body, "queries")
					}
					// The received schema also has to round-trip through the SDK input model.
					raw, err := json.Marshal(body)
					require.NoError(t, err)
					var decoded ledger.V2SchemaDataInput
					require.NoError(t, json.Unmarshal(raw, &decoded))
					require.Len(t, decoded.V2TransactionTemplates, len(sourceTemplates))
					require.Len(t, decoded.V2QueryTemplates, len(sourceQueries))
				case "/api/ledger/v2/demo-test/transactions":
					require.Equal(t, 1, installed)
					sent++
					require.Equal(t, schemaVersion, r.URL.Query().Get("schemaVersion"))
					require.Equal(t, *expected.IdempotencyKey, r.Header.Get("Idempotency-Key"))
					require.Equal(t, *expected.V2PostTransaction.Reference, body["reference"])
					require.False(t, seen[*expected.IdempotencyKey], "wire request identity must be unique")
					seen[*expected.IdempotencyKey] = true
					require.NotContains(t, body, "postings")
					require.NotContains(t, body, "force")
					metadata := body["metadata"].(map[string]any)
					require.Equal(t, name, metadata["demo.type"])
					require.Equal(t, "wire-run", metadata["demo.run"])
					script := body["script"].(map[string]any)
					vars := script["vars"].(map[string]any)
					require.Len(t, vars, len(expected.V2PostTransaction.Script.Vars))
					for key, value := range expected.V2PostTransaction.Script.Vars {
						require.Equal(t, value, vars[key], "Numscript inputs must remain strings")
					}
					if expected.V2PostTransaction.Script.Template != nil {
						template := *expected.V2PostTransaction.Script.Template
						require.Equal(t, template, script["template"])
						require.NotContains(t, script, "plain")
						require.Contains(t, sourceTemplates, template)
						require.Equal(t, template, metadata["demo.template"])
					} else {
						require.Equal(t, *expected.V2PostTransaction.Script.Plain, script["plain"])
						require.NotContains(t, script, "template")
						require.Equal(t, "DEMO_BOOTSTRAP", metadata["demo.template"])
						require.Equal(t, "experimental-interpreter", body["runtime"])
					}
					status = http.StatusOK
					response = fmt.Sprintf(`{"data":{"id":%d,"timestamp":"2026-11-16T00:00:00Z","postings":[],"reverted":false,"metadata":{},"reference":%q}}`, sent, *expected.V2PostTransaction.Reference)
				default:
					t.Fatalf("unexpected v2 request %s %s", r.Method, r.URL.Path)
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
			})}))
			createdResponse, err := sdk.Ledger.V2.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: "demo-test", V2CreateLedgerRequest: ledger.V2CreateLedgerRequest{V2Metadata: map[string]string{"demo.type": name, "demo.run": "wire-run"}}})
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, createdResponse.StatusCode)
			insertedResponse, err := sdk.Ledger.V2.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: "demo-test", Version: schemaVersion, V2SchemaData: schema})
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, insertedResponse.StatusCode)
			instance := newRecipeScenario(name, "worker_000")
			requests := 500
			if random, ok := instance.(*omnibus); ok {
				random.random = rand.New(rand.NewPCG(42, 99))
			}
			if planned, ok := instance.(*plannedScenario); ok {
				build := planned.build
				planned.build = func(c cycleContext) []recipeStep { c.amount = 10000; return build(c) }
				requests = len(planned.build(cycleContext{runID: "wire-run", namespace: "worker_000", amount: 10000}))
			}
			executed := map[string]bool{}
			for range requests {
				var commit func()
				expected, commit = instance.next("demo-test", "wire-run")
				response, err := sdk.Ledger.V2.CreateTransaction(t.Context(), expected)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				require.NotNil(t, response.V2CreateTransactionResponse)
				if expected.V2PostTransaction.Script.Template != nil {
					executed[*expected.V2PostTransaction.Script.Template] = true
				}
				commit()
			}
			require.Equal(t, 1, created)
			require.Equal(t, 1, installed)
			require.Equal(t, requests, sent)
			require.Len(t, executed, len(schema.V2TransactionTemplates), "a complete cycle must send every official template")
			t.Logf("v2 wire: %d templates, %d queries, %d transactions", len(sourceTemplates), len(sourceQueries), sent)
		})
	}
	require.Equal(t, 101, totalTemplates)
	require.Equal(t, 112, totalQueries)
}

func v2CatalogSourceSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	path := "recipes/" + name + ".yaml"

	raw, err := recipeFiles.ReadFile(path)
	require.NoError(t, err)
	var source any
	require.NoError(t, yaml.Unmarshal(raw, &source))
	normalized, err := json.Marshal(source)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(normalized, &document))
	return document
}
