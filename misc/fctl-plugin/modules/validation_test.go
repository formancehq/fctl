package modules_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/modules"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []commandCase{
		{name: "delete confirmation", command: "webhooks delete", args: []string{"id"}},
		{name: "unexpected body", command: "wallets show", flags: map[string]string{"id": "id"}, body: `{}`},
		{name: "route traversal", command: "webhooks delete", args: []string{".."}, flags: map[string]string{"confirm": "true"}},
		{name: "webhook userinfo", command: "webhooks create", args: []string{"https://user@example.invalid/hook", "event"}, flags: map[string]string{"confirm": "true"}},
		{name: "webhook scheme", command: "webhooks create", args: []string{"ftp://example.invalid/hook", "event"}, flags: map[string]string{"confirm": "true"}},
		{name: "empty event", command: "webhooks create", args: []string{"https://example.invalid/hook", " "}, flags: map[string]string{"confirm": "true"}},
		{name: "invalid secret", command: "webhooks change-secret", args: []string{"id", "not-base64"}, flags: map[string]string{"confirm": "true"}},
		{name: "reverse replay range", command: "webhooks deliveries replay-bulk", flags: map[string]string{"confirm": "true", "idempotency-key": "k", "created-at-from": "2026-02-01T00:00:00Z", "created-at-to": "2026-01-01T00:00:00Z"}},
		{name: "missing replay start", command: "webhooks deliveries replay-bulk", flags: map[string]string{"confirm": "true", "idempotency-key": "k"}},
		{name: "invalid replay status", command: "webhooks deliveries replay-bulk", flags: map[string]string{"confirm": "true", "idempotency-key": "k", "created-at-from": "2026-01-01T00:00:00Z", "status": "succeeded"}},
		{name: "invalid delivery status", command: "webhooks deliveries list", flags: map[string]string{"status": "other"}},
		{name: "invalid delivery timestamp", command: "webhooks deliveries list", flags: map[string]string{"created-at-from": "yesterday"}},
		{name: "zero page", command: "webhooks deliveries list", flags: map[string]string{"page-size": "0"}},
		{name: "empty stages", command: "orchestration workflows create", body: `{"stages":[]}`},
		{name: "scalar stage", command: "orchestration workflows create", body: `{"stages":[1]}`},
		{name: "null variables", command: "orchestration workflows run", args: []string{"flow"}, body: `null`},
		{name: "duplicate variables", command: "orchestration workflows run", args: []string{"flow"}, flags: map[string]string{"variable": "x=1,x=2"}},
		{name: "bad variables", command: "orchestration triggers create", args: []string{"event", "flow"}, flags: map[string]string{"vars": "x"}},
		{name: "duplicate event payload", command: "orchestration triggers test", args: []string{"trigger", `{}`}, body: `{}`},
		{name: "missing event payload", command: "orchestration triggers test", args: []string{"trigger"}},
		{name: "negative credit", command: "wallets credit", args: []string{"-1", "USD/2"}, flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{name: "fractional debit", command: "wallets debit", args: []string{"1.5", "USD/2"}, flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{name: "empty wallet asset", command: "wallets debit", args: []string{"1", " "}, flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{name: "invalid source", command: "wallets credit", args: []string{"1", "USD/2"}, flags: map[string]string{"id": "wallet", "source": "wallet=unknown:id", "confirm": "true"}},
		{name: "invalid destination", command: "wallets debit", args: []string{"1", "USD/2"}, flags: map[string]string{"id": "wallet", "destination": "account=", "confirm": "true"}},
		{name: "bad metadata", command: "wallets create", args: []string{"wallet"}, flags: map[string]string{"metadata": `{"amount":1}`, "confirm": "true"}},
		{name: "expired timestamp format", command: "wallets balances create", args: []string{"balance"}, flags: map[string]string{"id": "wallet", "expires-at": "tomorrow", "confirm": "true"}},
		{name: "negative hold amount", command: "wallets holds confirm", args: []string{"hold"}, flags: map[string]string{"amount": "-1", "confirm": "true"}},
		{name: "policy required ledger", command: "reconciliation policies create", body: `{"name":"policy","paymentsPoolID":"pool","ledgerQuery":{}}`, flags: map[string]string{"confirm": "true"}},
		{name: "policy required query", command: "reconciliation policies create", body: `{"name":"policy","ledgerName":"ledger","paymentsPoolID":"pool","ledgerQuery":null}`, flags: map[string]string{"confirm": "true"}},
		{name: "invalid reconciliation time", command: "reconciliation policies reconcile", args: []string{"policy", "yesterday", "2026-01-01T00:00:00Z"}},
		{name: "conflicting query", command: "reconciliation rules list", flags: map[string]string{"query": `{}`}, body: `{}`},
		{name: "query scalar", command: "reconciliation alerts list", flags: map[string]string{"query": `[]`}},
		{name: "negative margin", command: "reconciliation rules evaluate", args: []string{"rule"}, flags: map[string]string{"safety-margin": "-1s", "confirm": "true"}},
		{name: "invalid source PIT", command: "reconciliation rules evaluate", args: []string{"rule"}, flags: map[string]string{"source-pit": "ledger=yesterday", "confirm": "true"}},
		{name: "empty actor", command: "reconciliation alerts ack", args: []string{"alert"}, flags: map[string]string{"by": " ", "confirm": "true"}},
		{name: "empty acceptance note", command: "reconciliation alerts accept", args: []string{"alert"}, flags: map[string]string{"by": "qa", "note": " ", "confirm": "true"}},
		{name: "past snooze", command: "reconciliation alerts snooze", args: []string{"alert"}, flags: map[string]string{"by": "qa", "until": "2000-01-01T00:00:00Z", "confirm": "true"}},
		{name: "empty account payload", command: "payments accounts create", body: `{}`, flags: map[string]string{"confirm": "true"}},
		{name: "duplicate payment metadata", command: "payments payments set-metadata", args: []string{"id", "x=1", "x=2"}, flags: map[string]string{"confirm": "true"}},
		{name: "missing payment metadata", command: "payments payments set-metadata", args: []string{"id"}, flags: map[string]string{"confirm": "true"}},
		{name: "bad transfer status", command: "payments transfer-initiation update-status", args: []string{"id", "other"}, flags: map[string]string{"confirm": "true"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runFailure(t, tc, "") })
	}
}

func runFailure(t *testing.T, tc commandCase, want string) {
	t.Helper()
	server, verify := fixture(t, tc.exchanges)
	defer verify()
	path := strings.Fields(tc.command)
	p := modules.Factories()[path[0]](server.Client())
	var body json.RawMessage
	if tc.body != "" {
		body = json.RawMessage(tc.body)
	}
	_, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: path, Args: tc.args, Flags: tc.flags, Body: body, Endpoint: server.URL + "/gateway/service"})
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected failure containing %q, got %v", want, err)
	}
}

func TestPaymentsVersionAndSchemaFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tc            commandCase
		version, want string
	}{
		{commandCase{command: "payments orders list"}, "3.2.9", "3.3.0"},
		{commandCase{command: "payments pools update-query", args: []string{"id"}, body: `{"query":{}}`, flags: map[string]string{"confirm": "true"}}, "3.0.0", "3.1.0"},
		{commandCase{command: "payments tasks get", args: []string{"id"}}, "2.4.0", "3.0.0"},
		{commandCase{command: "payments transfer-initiation update-status", args: []string{"id", "VALIDATED"}, flags: map[string]string{"confirm": "true"}}, "3.4.8", "unavailable"},
		{commandCase{command: "payments payments list"}, "4.0.0", "unsupported"},
		{commandCase{command: "payments payments list"}, "development", "unrecognized"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","query":{}}`, flags: map[string]string{"confirm": "true"}}, "2.0.0", "dynamic pools"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","accountIDs":[""]}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "identifier"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","accountIDs":1}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "array"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","query":[]}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "query"},
		{commandCase{command: "payments bank-accounts create", body: `{"name":"bank"}`, flags: map[string]string{"confirm": "true"}}, "2.4.0", "country"},
		{commandCase{command: "payments pools create", body: `{"name":1}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "string"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","metadata":{"n":1}}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "metadata"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","validated":"true"}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "boolean"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","amount":-1}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "nonnegative"},
		{commandCase{command: "payments pools create", body: `{"name":"pool","createdAt":5}`, flags: map[string]string{"confirm": "true"}}, "3.4.8", "RFC3339"},
	}
	for _, tc := range cases {
		t.Run(tc.tc.command+" "+tc.version+" "+tc.want, func(t *testing.T) {
			t.Parallel()
			tc.tc.exchanges = []exchange{{method: "GET", path: "/_info", response: `{"version":"` + tc.version + `"}`}}
			runFailure(t, tc.tc, tc.want)
		})
	}
}

func TestDependentReadFailuresPreventMutation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tc   commandCase
		want string
	}{
		{commandCase{command: "wallets credit", args: []string{"1", "USD/2"}, flags: map[string]string{"name": "missing", "confirm": "true"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "name=missing&pageSize=2", response: `{"cursor":{"data":[]}}`}}}, "not found"},
		{commandCase{command: "wallets show", flags: map[string]string{"name": "duplicate"}, exchanges: []exchange{{method: "GET", path: "/wallets", query: "name=duplicate&pageSize=2", response: `{"cursor":{"data":[{"id":"1"},{"id":"2"}]}}`}}}, "ambiguous"},
		{commandCase{command: "orchestration instances show", args: []string{"id"}, exchanges: []exchange{{method: "GET", path: "/instances/id", response: `{"data":{}}`}}}, "workflowID"},
		{commandCase{command: "orchestration instances describe", args: []string{"id"}, exchanges: []exchange{{method: "GET", path: "/instances/id/history", response: `{"data":{}}`}}}, "data array"},
		{commandCase{command: "payments connectors install", args: []string{"DUMMYPAY"}, body: `{"name":"qa"}`, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "GET", path: "/_info", response: `{"version":"3.4.8"}`}, {method: "GET", path: "/v3/connectors/configs", response: `{"data":[]}`}}}, "missing a data object"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) { t.Parallel(); runFailure(t, tc.tc, tc.want) })
	}
}

func TestManifestIsOfflineAndIndependent(t *testing.T) {
	t.Parallel()
	for name, factory := range modules.Factories() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := factory(nil)
			manifest, err := p.GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Version != "1.0.0" || manifest.Service != name || manifest.Root.Target != "stack" {
				t.Fatalf("invalid manifest: %+v", manifest)
			}
			manifest.Root.Subcommands[0].Use = "modified"
			again, err := p.GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if again.Root.Subcommands[0].Use == "modified" {
				t.Fatal("manifest metadata aliases internal state")
			}
		})
	}
}

func TestCancellationPropagates(t *testing.T) {
	t.Parallel()
	server, verify := fixture(t, nil)
	defer verify()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	p := modules.Factories()["webhooks"](server.Client())
	_, err := p.Execute(ctx, pluginsdk.ExecuteRequest{CommandPath: []string{"webhooks", "list"}, Endpoint: server.URL + "/gateway/service"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestHistoricalUnderscoreAliases(t *testing.T) {
	t.Parallel()
	cases := []commandCase{
		{command: "payments bank_accounts get", args: []string{"id"}, exchanges: []exchange{{method: "GET", path: "/_info", response: `{"version":"3.4.8"}`}, {method: "GET", path: "/v3/bank-accounts/id"}}},
		{command: "payments transfer_initiation update_status", args: []string{"id", "VALIDATED"}, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "GET", path: "/_info", response: `{"version":"2.4.0"}`}, {method: "POST", path: "/transfer-initiations/id/status", body: `{"status":"VALIDATED"}`}}},
		{command: "payments payment_initiations approve", args: []string{"id"}, flags: map[string]string{"confirm": "true"}, exchanges: []exchange{{method: "GET", path: "/_info", response: `{"version":"3.4.8"}`}, {method: "POST", path: "/v3/payment-initiations/id/approve"}}},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}
