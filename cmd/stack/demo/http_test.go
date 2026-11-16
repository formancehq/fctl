package demo

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Exercise the actual generated SDK serializer and response parser, without
// authentication or a network connection to a deployed stack.
func TestGenerateThroughHTTPClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		var paths []string
		sdk := formance.New(formance.WithServerURL("https://demo.invalid"), formance.WithClient(&http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
			paths = append(paths, r.Method+" "+r.URL.Path)
			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, json.Unmarshal(raw, &body))
			status := 204
			response := ""
			switch r.URL.Path {
			case "/api/ledger/v2/demo-test":
				require.Equal(t, "omnibus", body["metadata"].(map[string]any)["demo.type"])
			case "/api/ledger/v2/demo-test/schemas/v1":
				templates := body["transactions"].(map[string]any)
				require.Len(t, templates, 5)
				for _, template := range templates {
					require.Equal(t, "experimental-interpreter", template.(map[string]any)["runtime"])
				}
			case "/api/ledger/v2/demo-test/transactions":
				require.Equal(t, "v1", r.URL.Query().Get("schemaVersion"))
				require.Equal(t, "test-0", r.Header.Get("Idempotency-Key"))
				require.Equal(t, "test-0", body["reference"])
				script := body["script"].(map[string]any)
				require.NotEmpty(t, script["template"])
				require.Equal(t, "EUR/2", script["vars"].(map[string]any)["asset"])
				status = 200
				response = `{"data":{"id":0,"timestamp":"2026-11-16T00:00:00Z","postings":[],"reverted":false,"metadata":{},"reference":"test-0"}}`
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
		})}))
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
		schema, err := omnibusSchema()
		require.NoError(t, err)
		require.NoError(t, c.generate(ctx, sdk.Ledger.V2, schema, 1, 500*time.Millisecond, &bytes.Buffer{}))
		require.EqualValues(t, 1, c.store.Transactions)
		require.Equal(t, []string{"POST /api/ledger/v2/demo-test", "POST /api/ledger/v2/demo-test/schemas/v1", "POST /api/ledger/v2/demo-test/transactions"}, paths)
	})
}
