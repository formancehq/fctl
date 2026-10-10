package modules_test

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/modules"
)

// These thirteen command signatures come from the pinned historical command
// tree. The host decodes the file and leaves its source argument in place.
func TestHistoricalFileArguments(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	account := `{"connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"qa","type":"INTERNAL"}`
	payment := `{"amount":9007199254740993,"asset":"USD/2","connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"qa","scheme":"OTHER","status":"SUCCEEDED","type":"TRANSFER"}`
	transfer := `{"amount":1,"asset":"USD/2","description":"qa","destinationAccountID":"dst","reference":"qa","scheduledAt":"2026-01-01T00:00:00Z","sourceAccountID":"src","type":"TRANSFER","validated":true}`
	reverse := `{"amount":1,"asset":"USD/2","description":"qa","metadata":{},"reference":"qa"}`
	cases := []commandCase{
		{command: "orchestration workflows create", body: `{"stages":[{"delay":{"duration":"1s"}}]}`, exchanges: []exchange{{method: "POST", path: "/workflows", body: `{"stages":[{"delay":{"duration":"1s"}}]}`}}},
		payCase("accounts create", nil, confirm, account, "3.4.8", exchange{method: "POST", path: "/accounts", body: account}),
		payCase("bank-accounts create", nil, confirm, `{"name":"qa"}`, "3.4.8", exchange{method: "POST", path: "/v3/bank-accounts", body: `{"name":"qa"}`}),
		payCase("payments create", nil, confirm, payment, "3.4.8", exchange{method: "POST", path: "/payments", body: payment}),
		payCase("pools create", nil, confirm, `{"name":"qa","accountIDs":[]}`, "3.4.8", exchange{method: "POST", path: "/v3/pools", body: `{"name":"qa","accountIDs":[]}`}),
		payCase("pools update-query", []string{"pool"}, confirm, `{"query":{"$match":{"type":"INTERNAL"}}}`, "3.4.8", exchange{method: "PATCH", path: "/v3/pools/pool/query", body: `{"query":{"$match":{"type":"INTERNAL"}}}`}),
		payCase("transfer-initiation create", nil, confirm, transfer, "3.4.8", exchange{method: "POST", path: "/transfer-initiations", body: transfer}),
		payCase("transfer-initiation reverse", []string{"transfer"}, confirm, reverse, "3.4.8", exchange{method: "POST", path: "/transfer-initiations/transfer/reverse", body: reverse}),
		payCase("connectors install", []string{"stripe"}, confirm, `{"name":"qa"}`, "3.4.8", exchange{method: "GET", path: "/v3/connectors/configs", response: `{"data":{"stripe":{}}}`}, exchange{method: "POST", path: "/v3/connectors/install/stripe", body: `{"name":"qa","provider":"stripe"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn"}, `{"name":"qa"}`, "3.4.8", exchange{method: "GET", path: "/v3/connectors/configs", response: `{"data":{"stripe":{}}}`}, exchange{method: "PATCH", path: "/v3/connectors/conn/config", body: `{"name":"qa","provider":"stripe"}`}),
		{command: "reconciliation policies create", flags: confirm, body: `{"name":"qa","ledgerName":"qa","paymentsPoolID":"pool","ledgerQuery":{}}`, exchanges: []exchange{{method: "POST", path: "/policies", body: `{"name":"qa","ledgerName":"qa","paymentsPoolID":"pool","ledgerQuery":{}}`}}},
		{command: "reconciliation rules create", flags: confirm, body: `{"name":"qa","templateKind":"ledger_invariant","templateSpec":{"terms":[],"tolerance":{"USD/2":0}}}`, exchanges: []exchange{{method: "POST", path: "/rules", body: `{"name":"qa","templateKind":"ledger_invariant","templateSpec":{"terms":[],"tolerance":{"USD/2":0}}}`}}},
		{command: "reconciliation rules update", args: []string{"rule"}, flags: confirm, body: `{"enabled":false}`, exchanges: []exchange{{method: "PATCH", path: "/rules/rule", body: `{"enabled":false}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()
			path := strings.Fields(tc.command)
			manifest, err := modules.Factories()[path[0]](nil).GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			command, err := pluginsdk.FindCommand(manifest, path)
			if err != nil {
				t.Fatal(err)
			}
			if command.Files == nil || command.Files.ReadArgument == nil || *command.Files.ReadArgument != len(tc.args) || command.Files.ReadFormat != "yaml" {
				t.Fatalf("missing historical file input: %+v", command.Files)
			}
			if command.Args.Min != len(tc.args) || command.Args.Max != len(tc.args)+1 || !strings.Contains(command.Use, "[<file>|-]") {
				t.Fatalf("wrong file signature: %+v", command)
			}
			// This deliberately nonexistent source proves the module itself never reads it.
			tc.args = append(append([]string{}, tc.args...), "/does-not-exist/fixture.yaml")
			runCase(t, tc)
		})
	}
}

func TestFileInputsOnlyOnHistoricalPayloadCommands(t *testing.T) {
	t.Parallel()
	cases := []string{"orchestration workflows run", "orchestration triggers test", "payments pools add-account", "reconciliation rules list", "wallets create", "webhooks create"}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			names := strings.Fields(path)
			manifest, err := modules.Factories()[names[0]](nil).GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			command, err := pluginsdk.FindCommand(manifest, names)
			if err != nil {
				t.Fatal(err)
			}
			if command.Files != nil {
				t.Fatal("non-file historical command gained a file source")
			}
		})
	}
}
