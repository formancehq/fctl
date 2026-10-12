package payments

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestHistoricalFileArguments(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	account := `{"connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"qa","type":"INTERNAL"}`
	payment := `{"amount":9007199254740993,"asset":"USD/2","connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"qa","scheme":"OTHER","status":"SUCCEEDED","type":"TRANSFER"}`
	transfer := `{"amount":1,"asset":"USD/2","description":"qa","destinationAccountID":"dst","reference":"qa","scheduledAt":"2026-01-01T00:00:00Z","sourceAccountID":"src","type":"TRANSFER","validated":true}`
	reverse := `{"amount":1,"asset":"USD/2","description":"qa","metadata":{},"reference":"qa"}`
	cases := []testutil.Case{
		payCase("accounts create", nil, confirm, account, "3.4.8", testutil.Exchange{Method: "POST", Path: "/accounts", Body: account}),
		payCase("bank-accounts create", nil, confirm, `{"name":"qa"}`, "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/bank-accounts", Body: `{"name":"qa"}`}),
		payCase("payments create", nil, confirm, payment, "3.4.8", testutil.Exchange{Method: "POST", Path: "/payments", Body: payment}),
		payCase("pools create", nil, confirm, `{"name":"qa","accountIDs":[]}`, "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/pools", Body: `{"name":"qa","accountIDs":[]}`}),
		payCase("pools update-query", []string{"pool"}, confirm, `{"query":{"$match":{"type":"INTERNAL"}}}`, "3.4.8", testutil.Exchange{Method: "PATCH", Path: "/v3/pools/pool/query", Body: `{"query":{"$match":{"type":"INTERNAL"}}}`}),
		payCase("transfer-initiation create", nil, confirm, transfer, "3.4.8", testutil.Exchange{Method: "POST", Path: "/transfer-initiations", Body: transfer}),
		payCase("transfer-initiation reverse", []string{"transfer"}, confirm, reverse, "3.4.8", testutil.Exchange{Method: "POST", Path: "/transfer-initiations/transfer/reverse", Body: reverse}),
		payCase("connectors install", []string{"stripe"}, confirm, `{"name":"qa"}`, "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs", Response: `{"data":{"stripe":{}}}`}, testutil.Exchange{Method: "POST", Path: "/v3/connectors/install/stripe", Body: `{"name":"qa","provider":"stripe"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn"}, `{"name":"qa"}`, "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs", Response: `{"data":{"stripe":{}}}`}, testutil.Exchange{Method: "PATCH", Path: "/v3/connectors/conn/config", Body: `{"name":"qa","provider":"stripe"}`}),
	}
	for _, tc := range cases {
		t.Run(tc.Command, func(t *testing.T) {
			t.Parallel()
			path := strings.Fields(tc.Command)
			manifest, err := New(nil).GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			command, err := pluginsdk.FindCommand(manifest, path)
			if err != nil {
				t.Fatal(err)
			}
			if command.Files == nil || command.Files.ReadArgument == nil || *command.Files.ReadArgument != len(tc.Args) || command.Files.ReadFormat != "yaml" {
				t.Fatalf("missing historical file input: %+v", command.Files)
			}
			if command.Args.Min != len(tc.Args) || command.Args.Max != len(tc.Args)+1 || !strings.Contains(command.Use, "[<file>|-]") {
				t.Fatalf("wrong file signature: %+v", command)
			}
			// This deliberately nonexistent source proves the module itself never reads it.
			tc.Args = append(append([]string{}, tc.Args...), "/does-not-exist/fixture.yaml")
			testutil.RunCase(t, New, tc)
		})
	}
}

func TestFileInputsOnlyOnHistoricalPayloadCommands(t *testing.T) {
	t.Parallel()
	cases := []string{"payments pools add-account"}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			names := strings.Fields(path)
			manifest, err := New(nil).GetManifest(t.Context())
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
