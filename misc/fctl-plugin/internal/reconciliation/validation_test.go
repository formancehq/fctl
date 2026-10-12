package reconciliation

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "policy required ledger", Command: "reconciliation policies create", Body: `{"name":"policy","paymentsPoolID":"pool","ledgerQuery":{}}`, Flags: map[string]string{"confirm": "true"}},
		{Name: "policy required query", Command: "reconciliation policies create", Body: `{"name":"policy","ledgerName":"ledger","paymentsPoolID":"pool","ledgerQuery":null}`, Flags: map[string]string{"confirm": "true"}},
		{Name: "invalid reconciliation time", Command: "reconciliation policies reconcile", Args: []string{"policy", "yesterday", "2026-01-01T00:00:00Z"}},
		{Name: "conflicting query", Command: "reconciliation rules list", Flags: map[string]string{"query": `{}`}, Body: `{}`},
		{Name: "query scalar", Command: "reconciliation alerts list", Flags: map[string]string{"query": `[]`}},
		{Name: "negative margin", Command: "reconciliation rules evaluate", Args: []string{"rule"}, Flags: map[string]string{"safety-margin": "-1s", "confirm": "true"}},
		{Name: "invalid source PIT", Command: "reconciliation rules evaluate", Args: []string{"rule"}, Flags: map[string]string{"source-pit": "ledger=yesterday", "confirm": "true"}},
		{Name: "empty actor", Command: "reconciliation alerts ack", Args: []string{"alert"}, Flags: map[string]string{"by": " ", "confirm": "true"}},
		{Name: "empty acceptance note", Command: "reconciliation alerts accept", Args: []string{"alert"}, Flags: map[string]string{"by": "qa", "note": " ", "confirm": "true"}},
		{Name: "past snooze", Command: "reconciliation alerts snooze", Args: []string{"alert"}, Flags: map[string]string{"by": "qa", "until": "2000-01-01T00:00:00Z", "confirm": "true"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFailure(t, New, tc, "")
		})
	}
}
