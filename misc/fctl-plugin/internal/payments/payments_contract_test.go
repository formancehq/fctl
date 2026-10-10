package payments

import (
	"fmt"
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func payCase(command string, args []string, flags map[string]string, body, version string, steps ...testutil.Exchange) testutil.Case {
	return testutil.Case{Name: command + " " + version, Command: "payments " + command, Args: args, Flags: flags, Body: body,
		Exchanges: append([]testutil.Exchange{{Method: "GET", Path: "/_info", Response: fmt.Sprintf(`{"data":{"version":%q}}`, version)}}, steps...)}
}

func TestPaymentsHistoricalContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	account := `{"connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"ref","type":"INTERNAL"}`
	payment := `{"amount":9007199254740993,"asset":"USD/2","connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"ref","scheme":"other","status":"SUCCEEDED","type":"PAYOUT"}`
	initiation := `{"amount":9007199254740993,"asset":"USD/2","description":"transfer","destinationAccountID":"dst","reference":"ref","scheduledAt":"2026-01-01T00:00:00Z","sourceAccountID":"src","type":"TRANSFER","validated":false}`
	reversal := `{"amount":9007199254740993,"asset":"USD/2","description":"reverse","metadata":{"reason":"test"},"reference":"reverse-ref"}`
	cases := []testutil.Case{
		payCase("accounts create", nil, confirm, account, "3.4.8", testutil.Exchange{Method: "POST", Path: "/accounts", Body: account}),
		payCase("accounts get", []string{"a/%"}, nil, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/accounts/a%2F%25"}),
		payCase("accounts list", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/accounts", Query: "pageSize=100"}),
		payCase("accounts balances", []string{"a"}, map[string]string{"cursor": "next"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/accounts/a/balances", Query: "cursor=next"}),
		payCase("payments create", nil, confirm, payment, "3.4.8", testutil.Exchange{Method: "POST", Path: "/payments", Body: payment}),
		payCase("payments get", []string{"p"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/payments/p"}),
		payCase("payments list", nil, map[string]string{"page-size": "2"}, "", "2.5.0", testutil.Exchange{Method: "GET", Path: "/payments", Query: "pageSize=2"}),
		payCase("payments set-metadata", []string{"p", "key=value=extra"}, confirm, "", "3.4.8", testutil.Exchange{Method: "PATCH", Path: "/payments/p/metadata", Body: `{"key":"value=extra"}`}),
		payCase("bank-accounts create", nil, confirm, `{"name":"bank"}`, "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/bank-accounts", Body: `{"name":"bank"}`}),
		payCase("bank-accounts create", nil, confirm, `{"name":"bank","country":"FR"}`, "2.5.0", testutil.Exchange{Method: "POST", Path: "/bank-accounts", Body: `{"name":"bank","country":"FR"}`}),
		payCase("bank-accounts get", []string{"b"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/bank-accounts/b"}),
		payCase("bank-accounts get", []string{"b"}, nil, "", "2.5.0", testutil.Exchange{Method: "GET", Path: "/bank-accounts/b"}),
		payCase("bank-accounts list", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/bank-accounts", Query: "pageSize=100"}),
		payCase("bank-accounts forward", []string{"b", "conn"}, confirm, "", "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/bank-accounts/b/forward", Body: `{"connectorID":"conn"}`}),
		payCase("bank-accounts forward", []string{"b", "conn"}, confirm, "", "2.5.0", testutil.Exchange{Method: "POST", Path: "/bank-accounts/b/forward", Body: `{"connectorID":"conn"}`}),
		payCase("bank-accounts update-metadata", []string{"b"}, map[string]string{"confirm": "true", "metadata": "key=value"}, "", "3.4.8", testutil.Exchange{Method: "PATCH", Path: "/v3/bank-accounts/b/metadata", Body: `{"metadata":{"key":"value"}}`}),
		payCase("pools create", nil, confirm, `{"name":"pool","accountIDs":["a"]}`, "2.0.0", testutil.Exchange{Method: "POST", Path: "/pools", Body: `{"name":"pool","accountIDs":["a"]}`}),
		payCase("pools create", nil, confirm, `{"name":"pool","query":{"$match":{"amount":9007199254740993}}}`, "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/pools", Body: `{"name":"pool","query":{"$match":{"amount":9007199254740993}}}`}),
		payCase("pools get", []string{"pool"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/pools/pool"}),
		payCase("pools list", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/pools", Query: "pageSize=100"}),
		payCase("pools delete", []string{"pool"}, confirm, "", "3.4.8", testutil.Exchange{Method: "DELETE", Path: "/pools/pool", Status: 204}),
		payCase("pools balances", []string{"pool", "2026-01-01T00:00:00Z"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/pools/pool/balances", Query: "at=2026-01-01T00%3A00%3A00Z"}),
		payCase("pools latest-balances", []string{"pool"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/pools/pool/balances/latest"}),
		payCase("pools latest-balances", []string{"pool"}, nil, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/pools/pool/balances/latest"}),
		payCase("pools add-account", []string{"pool", "a"}, nil, "", "3.4.8", testutil.Exchange{Method: "POST", Path: "/pools/pool/accounts", Body: `{"accountID":"a"}`}),
		payCase("pools remove-account", []string{"pool", "a"}, confirm, "", "3.4.8", testutil.Exchange{Method: "DELETE", Path: "/pools/pool/accounts/a"}),
		payCase("pools update-query", []string{"pool"}, confirm, `{"query":{"$match":{"type":"INTERNAL"}}}`, "3.4.8", testutil.Exchange{Method: "PATCH", Path: "/v3/pools/pool/query", Body: `{"query":{"$match":{"type":"INTERNAL"}}}`}),
		payCase("transfer-initiation create", nil, confirm, initiation, "3.4.8", testutil.Exchange{Method: "POST", Path: "/transfer-initiations", Body: initiation}),
		payCase("transfer-initiation get", []string{"ti"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/transfer-initiations/ti"}),
		payCase("transfer-initiation list", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/transfer-initiations", Query: "pageSize=100"}),
		payCase("transfer-initiation approve", []string{"ti"}, confirm, "", "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/payment-initiations/ti/approve"}),
		payCase("transfer-initiation reject", []string{"ti"}, confirm, "", "3.4.8", testutil.Exchange{Method: "POST", Path: "/v3/payment-initiations/ti/reject"}),
		payCase("transfer-initiation retry", []string{"ti"}, confirm, "", "3.4.8", testutil.Exchange{Method: "POST", Path: "/transfer-initiations/ti/retry"}),
		payCase("transfer-initiation reverse", []string{"ti"}, confirm, reversal, "3.4.8", testutil.Exchange{Method: "POST", Path: "/transfer-initiations/ti/reverse", Body: reversal}),
		payCase("transfer-initiation delete", []string{"ti"}, confirm, "", "3.4.8", testutil.Exchange{Method: "DELETE", Path: "/transfer-initiations/ti"}),
		payCase("transfer-initiation update-status", []string{"ti", "VALIDATED"}, confirm, "", "2.0.0", testutil.Exchange{Method: "POST", Path: "/transfer-initiations/ti/status", Body: `{"status":"VALIDATED"}`}),
		payCase("tasks get", []string{"task"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/tasks/task"}),
		payCase("orders get", []string{"order"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/orders/order"}),
		payCase("conversions get", []string{"conv"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/conversions/conv"}),
		payCase("orders list", nil, map[string]string{"connector-id": "c", "reference": "ref", "direction": "BUY", "status": "FILLED", "type": "MARKET", "source-asset": "USD/2", "destination-asset": "BTC/8", "page-size": "2"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/orders", Query: "pageSize=2", Body: `{"$and":[{"$match":{"connector_id":"c"}},{"$match":{"reference":"ref"}},{"$match":{"direction":"BUY"}},{"$match":{"status":"FILLED"}},{"$match":{"type":"MARKET"}},{"$match":{"source_asset":"USD/2"}},{"$match":{"destination_asset":"BTC/8"}}]}`}),
		payCase("conversions list", nil, map[string]string{"cursor": "next", "reference": "ignored-on-next-page"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/conversions", Query: "cursor=next"}),
		payCase("conversions list", nil, map[string]string{"status": "COMPLETED"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/conversions", Query: "pageSize=100", Body: `{"$and":[{"$match":{"status":"COMPLETED"}}]}`}),
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
