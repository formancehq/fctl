package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

// Explicitly opt in: this creates and retains a separate demo ledger per recipe.
// Ordinary test runs remain offline and never load the user's profile.
func TestCatalogLive(t *testing.T) {
	stack := os.Getenv("FCTL_DEMO_LIVE_STACK")
	organization := os.Getenv("FCTL_DEMO_LIVE_ORGANIZATION")
	if stack == "" || organization == "" {
		t.Skip("set FCTL_DEMO_LIVE_STACK and FCTL_DEMO_LIVE_ORGANIZATION to opt in to writes")
	}
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	cmd := NewCommand()
	cmd.Flags().String(fctl.ConfigDir, filepath.Join(home, ".config", "formance", "fctl"), "")
	cmd.Flags().String(fctl.ProfileFlag, os.Getenv("FCTL_DEMO_LIVE_PROFILE"), "")
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"--organization", organization, "--stack", stack}))
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	require.NoError(t, err)
	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	require.NoError(t, err)
	major, err := detectLedgerMajor(t.Context(), clients.HTTPClient, clients.URI)
	require.NoError(t, err)
	if expected := os.Getenv("FCTL_DEMO_LIVE_MAJOR"); expected != "" {
		require.Equal(t, expected, major, "refuse writes to an unexpected Ledger API")
	}
	for _, name := range recipeNames() {
		t.Run(name, func(t *testing.T) {
			runID := ksuid.New().String()
			ledgerName := "demo-" + name + "-" + runID
			schema, err := loadRecipeSchema(name)
			require.NoError(t, err)
			client, err := demoLedgerClient(t.Context(), clients, schema)
			require.NoError(t, err)
			t.Logf("creating ledger=%s API=%s", ledgerName, major)
			created, err := client.CreateLedger(t.Context(), operations.V2CreateLedgerRequest{Ledger: ledgerName, V2CreateLedgerRequest: ledger.V2CreateLedgerRequest{V2Metadata: map[string]string{"demo.run": runID, "demo.type": name, "demo.validation": "catalog"}}}, operations.WithOperationTimeout(30*time.Second))
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, created.StatusCode)
			inserted, err := client.InsertSchema(t.Context(), operations.V2InsertSchemaRequest{Ledger: ledgerName, Version: schemaVersion, V2SchemaData: schema}, operations.WithOperationTimeout(30*time.Second))
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, inserted.StatusCode)
			scenario := newRecipeScenario(name, "")
			if o, ok := scenario.(*omnibus); ok {
				o.random = rand.New(rand.NewPCG(42, 99))
			}
			seen := map[string]bool{}
			count := 0
			for count < 2000 {
				request, commit := scenario.next(ledgerName, runID)
				response, err := client.CreateTransaction(t.Context(), request, operations.WithOperationTimeout(30*time.Second))
				require.NoError(t, err, "ledger=%s step=%d template=%s", ledgerName, count, request.V2PostTransaction.V2Metadata["demo.template"])
				require.Equal(t, http.StatusOK, response.StatusCode)
				commit()
				count++
				if request.V2PostTransaction.Script.Template != nil {
					seen[*request.V2PostTransaction.Script.Template] = true
				}
				if plan, ok := scenario.(*plannedScenario); ok {
					if plan.cycle >= 2 {
						break
					}
				} else if len(seen) == len(schema.V2TransactionTemplates) {
					break
				}
			}
			require.Len(t, seen, len(schema.V2TransactionTemplates), "every official template must execute")
			readCount, readTemplates := readLiveTransactions(t, clients, major, ledgerName)
			require.Equal(t, count, readCount, "verify acknowledged writes by a fresh read")
			for template := range schema.V2TransactionTemplates {
				require.True(t, readTemplates[template], "template %s missing from committed transactions", template)
			}
			t.Logf("PASS API=%s recipe=%s ledger=%s transactions=%d templates=%d/%d", major, name, ledgerName, count, len(seen), len(schema.V2TransactionTemplates))
		})
	}
}

func readLiveTransactions(t *testing.T, clients *fctl.StackClients, major, ledgerName string) (int, map[string]bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(clients.URI, "/")+"/api/ledger/"+major+"/"+ledgerName+"/transactions?pageSize=1000", nil)
	require.NoError(t, err)
	response, err := clients.HTTPClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, "%s", string(body))
	type row struct {
		Metadata map[string]json.RawMessage `json:"metadata"`
	}
	var page struct {
		Data    []row `json:"data"`
		HasMore bool  `json:"hasMore"`
		Cursor  struct {
			Data    []row `json:"data"`
			HasMore bool  `json:"hasMore"`
		} `json:"cursor"`
	}
	require.NoError(t, json.Unmarshal(body, &page))
	require.False(t, page.HasMore || page.Cursor.HasMore, "validation writes must fit in one bounded page")
	rows := page.Data
	if rows == nil {
		rows = page.Cursor.Data
	}
	templates := map[string]bool{}
	for _, row := range rows {
		var template string
		require.NoError(t, json.Unmarshal(row.Metadata["demo.template"], &template), fmt.Sprint(row.Metadata))
		templates[template] = true
	}
	return len(rows), templates
}
