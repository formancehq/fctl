package modules_test

import "testing"

func TestWalletContracts(t *testing.T) {
	t.Parallel()
	cases := []commandCase{
		{name: "create", command: "wallets create", args: []string{"Alice"}, flags: map[string]string{"confirm": "true", "metadata": "customer=42", "ik": "create-1"}, exchanges: []exchange{{method: "POST", path: "/wallets", key: "create-1", body: `{"name":"Alice","metadata":{"customer":"42"}}`}}},
		{name: "update", command: "wallets update", args: []string{"w/1"}, flags: map[string]string{"confirm": "true", "metadata": "{\"customer\":\"43\"}", "ik": "update-1"}, exchanges: []exchange{{method: "PATCH", path: "/wallets/w%2F1", key: "update-1", body: `{"metadata":{"customer":"43"}}`}}},
		{name: "metadata list", command: "wallets list", flags: map[string]string{"name": "Alice", "metadata": "customer=42", "page-size": "2"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "name=Alice&metadata%5Bcustomer%5D=42&pageSize=2"}}},
		{name: "cursor list", command: "wallets list", flags: map[string]string{"cursor": "opaque +/=", "metadata": "customer=42"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "cursor=opaque+%2B%2F%3D"}}},
		{name: "show id", command: "wallets show", flags: map[string]string{"id": "w/1"}, exchanges: []exchange{{method: "GET", path: "/wallets/w%2F1"}}},
		{name: "show name", command: "wallets show", flags: map[string]string{"name": "Alice"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "name=Alice&pageSize=2", response: `{"cursor":{"data":[{"id":"w/1"}],"hasMore":false}}`}, {method: "GET", path: "/wallets/w%2F1"}}},
		{name: "create balance", command: "wallets balances create", args: []string{"promo"}, flags: map[string]string{"confirm": "true", "id": "w", "expires-at": "2027-01-01T00:00:00Z", "priority": "-9007199254740993"}, exchanges: []exchange{{method: "POST", path: "/wallets/w/balances", body: `{"name":"promo","expiresAt":"2027-01-01T00:00:00Z","priority":-9007199254740993}`}}},
		{name: "list balances", command: "wallets balances list", flags: map[string]string{"id": "w", "page-size": "2"}, exchanges: []exchange{{method: "GET", path: "/wallets/w/balances", query: "pageSize=2"}}},
		{name: "show balance", command: "wallets balances show", args: []string{"promo/1"}, flags: map[string]string{"id": "w"}, exchanges: []exchange{{method: "GET", path: "/wallets/w/balances/promo%2F1"}}},
		{name: "credit exact integer and subjects", command: "wallets credit", args: []string{"9007199254740993", "USD/2"}, flags: map[string]string{"confirm": "true", "id": "w", "source": "account=world,wallet=name:Bob/promo", "balance": "main", "metadata": "reference=credit-1", "ik": "credit-1"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "name=Bob&pageSize=2", response: `{"cursor":{"data":[{"id":"bob"}],"hasMore":false}}`}, {method: "POST", path: "/wallets/w/credit", key: "credit-1", body: `{"amount":{"amount":9007199254740993,"asset":"USD/2"},"balance":"main","metadata":{"reference":"credit-1"},"sources":[{"type":"ACCOUNT","identifier":"world"},{"type":"WALLET","identifier":"bob","balance":"promo"}]}`}}},
		{name: "debit pending", command: "wallets debit", args: []string{"100", "USD/2"}, flags: map[string]string{"confirm": "true", "id": "w", "pending": "true", "balance": "main,promo", "destination": "wallet=id:target", "description": "pending debit", "ik": "debit-1"}, exchanges: []exchange{{method: "POST", path: "/wallets/w/debit", key: "debit-1", body: `{"amount":{"amount":100,"asset":"USD/2"},"pending":true,"balances":["main","promo"],"metadata":{},"description":"pending debit","destination":{"type":"WALLET","identifier":"target","balance":"main"}}`}}},
		{name: "holds list", command: "wallets holds list", flags: map[string]string{"id": "w", "metadata": "customer=42"}, exchanges: []exchange{{method: "GET", path: "/holds", query: "walletID=w&metadata%5Bcustomer%5D=42&pageSize=100"}}},
		{name: "hold show", command: "wallets holds show", args: []string{"h"}, exchanges: []exchange{{method: "GET", path: "/holds/h"}}},
		{name: "confirm hold", command: "wallets holds confirm", args: []string{"h"}, flags: map[string]string{"confirm": "true", "amount": "9007199254740993", "final": "true", "ik": "confirm-1"}, exchanges: []exchange{{method: "POST", path: "/holds/h/confirm", key: "confirm-1", body: `{"amount":9007199254740993,"final":true}`}}},
		{name: "void hold", command: "wallets holds void", args: []string{"h"}, flags: map[string]string{"confirm": "true", "ik": "void-1"}, exchanges: []exchange{{method: "POST", path: "/holds/h/void", key: "void-1"}}},
		{name: "transactions list", command: "wallets transactions list", flags: map[string]string{"id": "w"}, exchanges: []exchange{{method: "GET", path: "/transactions", query: "walletID=w&pageSize=100"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}

func TestReconciliationContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	cases := []commandCase{
		{name: "list", command: "reconciliation list", flags: map[string]string{"page-size": "2"}, exchanges: []exchange{{method: "GET", path: "/reconciliations", query: "pageSize=2"}}},
		{name: "get", command: "reconciliation get", args: []string{"r/1"}, exchanges: []exchange{{method: "GET", path: "/reconciliations/r%2F1"}}},
		{name: "policies list", command: "reconciliation policies list", flags: map[string]string{"cursor": "next"}, exchanges: []exchange{{method: "GET", path: "/policies", query: "cursor=next"}}},
		{name: "policy get", command: "reconciliation policies get", args: []string{"p"}, exchanges: []exchange{{method: "GET", path: "/policies/p"}}},
		{name: "create policy", command: "reconciliation policies create", flags: confirm, body: `{"name":"p","ledgerName":"book","paymentsPoolID":"pool","ledgerQuery":{"$match":{"amount":9007199254740993}}}`, exchanges: []exchange{{method: "POST", path: "/policies", body: `{"name":"p","ledgerName":"book","paymentsPoolID":"pool","ledgerQuery":{"$match":{"amount":9007199254740993}}}`}}},
		{name: "delete policy", command: "reconciliation policies delete", args: []string{"p"}, flags: confirm, exchanges: []exchange{{method: "DELETE", path: "/policies/p", status: 204}}},
		{name: "reconcile", command: "reconciliation policies reconcile", args: []string{"p", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"}, exchanges: []exchange{{method: "POST", path: "/policies/p/reconciliation", body: `{"reconciledAtLedger":"2026-01-01T00:00:00Z","reconciledAtPayments":"2026-01-02T00:00:00Z"}`}}},
		{name: "rules query", command: "reconciliation rules list", flags: map[string]string{"query": `{"$match":{"enabled":true,"amount":9007199254740993}}`}, exchanges: []exchange{{method: "GET", path: "/rules", query: "pageSize=100", body: `{"$match":{"enabled":true,"amount":9007199254740993}}`}}},
		{name: "rule get", command: "reconciliation rules get", args: []string{"rule"}, exchanges: []exchange{{method: "GET", path: "/rules/rule"}}},
		{name: "rule create", command: "reconciliation rules create", flags: confirm, body: `{"name":"rule","templateSpec":{"amount":9007199254740993}}`, exchanges: []exchange{{method: "POST", path: "/rules", body: `{"name":"rule","templateSpec":{"amount":9007199254740993}}`}}},
		{name: "rule patch false", command: "reconciliation rules update", args: []string{"rule"}, flags: confirm, body: `{"enabled":false,"labels":{}}`, exchanges: []exchange{{method: "PATCH", path: "/rules/rule", body: `{"enabled":false,"labels":{}}`}}},
		{name: "rule delete", command: "reconciliation rules delete", args: []string{"rule"}, flags: confirm, exchanges: []exchange{{method: "DELETE", path: "/rules/rule"}}},
		{name: "rule evaluate", command: "reconciliation rules evaluate", args: []string{"rule"}, flags: map[string]string{"confirm": "true", "at": "2026-01-01T00:00:00Z", "safety-margin": "30s", "source-pit": "ledger=2026-01-01T00:00:00Z,payments=2026-01-02T00:00:00Z"}, exchanges: []exchange{{method: "POST", path: "/rules/rule/evaluate", body: `{"at":"2026-01-01T00:00:00Z","safetyMargin":"30s","sourcePITs":{"ledger":"2026-01-01T00:00:00Z","payments":"2026-01-02T00:00:00Z"}}`}}},
		{name: "evaluations query", command: "reconciliation evaluations list", body: `{"$match":{"ruleID":"rule"}}`, exchanges: []exchange{{method: "GET", path: "/evaluations", query: "pageSize=100", body: `{"$match":{"ruleID":"rule"}}`}}},
		{name: "evaluation get", command: "reconciliation evaluations get", args: []string{"ev"}, exchanges: []exchange{{method: "GET", path: "/evaluations/ev"}}},
		{name: "alerts cursor", command: "reconciliation alerts list", flags: map[string]string{"cursor": "next", "query": `{"$match":{"status":"open"}}`}, exchanges: []exchange{{method: "GET", path: "/alerts", query: "cursor=next"}}},
		{name: "alert get", command: "reconciliation alerts get", args: []string{"al"}, exchanges: []exchange{{method: "GET", path: "/alerts/al"}}},
		{name: "alert events", command: "reconciliation alerts events", args: []string{"al"}, flags: map[string]string{"page-size": "10"}, exchanges: []exchange{{method: "GET", path: "/alerts/al/events", query: "pageSize=10"}}},
		{name: "ack", command: "reconciliation alerts ack", args: []string{"al"}, flags: map[string]string{"confirm": "true", "by": "engineer", "note": "investigating"}, exchanges: []exchange{{method: "POST", path: "/alerts/al/ack", body: `{"by":"engineer","note":"investigating"}`}}},
		{name: "resolve", command: "reconciliation alerts resolve", args: []string{"al"}, flags: map[string]string{"confirm": "true", "by": "engineer", "transaction-ref": "ref1,ref2"}, exchanges: []exchange{{method: "POST", path: "/alerts/al/resolve", body: `{"by":"engineer","transactionRefs":["ref1","ref2"]}`}}},
		{name: "accept", command: "reconciliation alerts accept", args: []string{"al"}, flags: map[string]string{"confirm": "true", "by": "engineer", "note": "approved adjustment"}, exchanges: []exchange{{method: "POST", path: "/alerts/al/accept", body: `{"by":"engineer","note":"approved adjustment"}`}}},
		{name: "snooze", command: "reconciliation alerts snooze", args: []string{"al"}, flags: map[string]string{"confirm": "true", "by": "engineer", "until": "2099-01-01T00:00:00Z"}, exchanges: []exchange{{method: "POST", path: "/alerts/al/snooze", body: `{"by":"engineer","until":"2099-01-01T00:00:00Z"}`}}},
		{name: "unsnooze", command: "reconciliation alerts unsnooze", args: []string{"al"}, flags: map[string]string{"confirm": "true", "by": "engineer"}, exchanges: []exchange{{method: "POST", path: "/alerts/al/unsnooze", body: `{"by":"engineer"}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}
