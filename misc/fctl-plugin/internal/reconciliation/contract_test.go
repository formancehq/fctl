package reconciliation

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestReconciliationContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	cases := []testutil.Case{
		{Name: "list", Command: "reconciliation list", Flags: map[string]string{"page-size": "2"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/reconciliations", Query: "pageSize=2"}}},
		{Name: "get", Command: "reconciliation get", Args: []string{"r/1"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/reconciliations/r%2F1"}}},
		{Name: "policies list", Command: "reconciliation policies list", Flags: map[string]string{"cursor": "next"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/policies", Query: "cursor=next"}}},
		{Name: "policy get", Command: "reconciliation policies get", Args: []string{"p"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/policies/p"}}},
		{Name: "create policy", Command: "reconciliation policies create", Flags: confirm, Body: `{"name":"p","ledgerName":"book","paymentsPoolID":"pool","ledgerQuery":{"$match":{"amount":9007199254740993}}}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/policies", Body: `{"name":"p","ledgerName":"book","paymentsPoolID":"pool","ledgerQuery":{"$match":{"amount":9007199254740993}}}`}}},
		{Name: "delete policy", Command: "reconciliation policies delete", Args: []string{"p"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "DELETE", Path: "/policies/p", Status: 204}}},
		{Name: "reconcile", Command: "reconciliation policies reconcile", Args: []string{"p", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/policies/p/reconciliation", Body: `{"reconciledAtLedger":"2026-01-01T00:00:00Z","reconciledAtPayments":"2026-01-02T00:00:00Z"}`}}},
		{Name: "rules query", Command: "reconciliation rules list", Flags: map[string]string{"query": `{"$match":{"enabled":true,"amount":9007199254740993}}`}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/rules", Query: "pageSize=100", Body: `{"$match":{"enabled":true,"amount":9007199254740993}}`}}},
		{Name: "rule get", Command: "reconciliation rules get", Args: []string{"rule"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/rules/rule"}}},
		{Name: "rule create", Command: "reconciliation rules create", Flags: confirm, Body: `{"name":"rule","templateSpec":{"amount":9007199254740993}}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/rules", Body: `{"name":"rule","templateSpec":{"amount":9007199254740993}}`}}},
		{Name: "rule patch false", Command: "reconciliation rules update", Args: []string{"rule"}, Flags: confirm, Body: `{"enabled":false,"labels":{}}`, Exchanges: []testutil.Exchange{{Method: "PATCH", Path: "/rules/rule", Body: `{"enabled":false,"labels":{}}`}}},
		{Name: "rule delete", Command: "reconciliation rules delete", Args: []string{"rule"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "DELETE", Path: "/rules/rule"}}},
		{Name: "rule evaluate", Command: "reconciliation rules evaluate", Args: []string{"rule"}, Flags: map[string]string{"confirm": "true", "at": "2026-01-01T00:00:00Z", "safety-margin": "30s", "source-pit": "ledger=2026-01-01T00:00:00Z,payments=2026-01-02T00:00:00Z"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/rules/rule/evaluate", Body: `{"at":"2026-01-01T00:00:00Z","safetyMargin":"30s","sourcePITs":{"ledger":"2026-01-01T00:00:00Z","payments":"2026-01-02T00:00:00Z"}}`}}},
		{Name: "evaluations query", Command: "reconciliation evaluations list", Body: `{"$match":{"ruleID":"rule"}}`, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/evaluations", Query: "pageSize=100", Body: `{"$match":{"ruleID":"rule"}}`}}},
		{Name: "evaluation get", Command: "reconciliation evaluations get", Args: []string{"ev"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/evaluations/ev"}}},
		{Name: "alerts cursor", Command: "reconciliation alerts list", Flags: map[string]string{"cursor": "next", "query": `{"$match":{"status":"open"}}`}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/alerts", Query: "cursor=next"}}},
		{Name: "alert get", Command: "reconciliation alerts get", Args: []string{"al"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/alerts/al"}}},
		{Name: "alert events", Command: "reconciliation alerts events", Args: []string{"al"}, Flags: map[string]string{"page-size": "10"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/alerts/al/events", Query: "pageSize=10"}}},
		{Name: "ack", Command: "reconciliation alerts ack", Args: []string{"al"}, Flags: map[string]string{"confirm": "true", "by": "engineer", "note": "investigating"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/alerts/al/ack", Body: `{"by":"engineer","note":"investigating"}`}}},
		{Name: "resolve", Command: "reconciliation alerts resolve", Args: []string{"al"}, Flags: map[string]string{"confirm": "true", "by": "engineer", "transaction-ref": "ref1,ref2"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/alerts/al/resolve", Body: `{"by":"engineer","transactionRefs":["ref1","ref2"]}`}}},
		{Name: "accept", Command: "reconciliation alerts accept", Args: []string{"al"}, Flags: map[string]string{"confirm": "true", "by": "engineer", "note": "approved adjustment"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/alerts/al/accept", Body: `{"by":"engineer","note":"approved adjustment"}`}}},
		{Name: "snooze", Command: "reconciliation alerts snooze", Args: []string{"al"}, Flags: map[string]string{"confirm": "true", "by": "engineer", "until": "2099-01-01T00:00:00Z"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/alerts/al/snooze", Body: `{"by":"engineer","until":"2099-01-01T00:00:00Z"}`}}},
		{Name: "unsnooze", Command: "reconciliation alerts unsnooze", Args: []string{"al"}, Flags: map[string]string{"confirm": "true", "by": "engineer"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/alerts/al/unsnooze", Body: `{"by":"engineer"}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
