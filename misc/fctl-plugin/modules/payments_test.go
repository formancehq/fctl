package modules_test

import (
	"fmt"
	"testing"
)

func payCase(command string, args []string, flags map[string]string, body, version string, steps ...exchange) commandCase {
	return commandCase{name: command + " " + version, command: "payments " + command, args: args, flags: flags, body: body,
		exchanges: append([]exchange{{method: "GET", path: "/_info", response: fmt.Sprintf(`{"data":{"version":%q}}`, version)}}, steps...)}
}

func TestPaymentsHistoricalContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	account := `{"connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"ref","type":"INTERNAL"}`
	payment := `{"amount":9007199254740993,"asset":"USD/2","connectorID":"conn","createdAt":"2026-01-01T00:00:00Z","reference":"ref","scheme":"other","status":"SUCCEEDED","type":"PAYOUT"}`
	initiation := `{"amount":9007199254740993,"asset":"USD/2","description":"transfer","destinationAccountID":"dst","reference":"ref","scheduledAt":"2026-01-01T00:00:00Z","sourceAccountID":"src","type":"TRANSFER","validated":false}`
	reversal := `{"amount":9007199254740993,"asset":"USD/2","description":"reverse","metadata":{"reason":"test"},"reference":"reverse-ref"}`
	cases := []commandCase{
		payCase("accounts create", nil, confirm, account, "3.4.8", exchange{method: "POST", path: "/accounts", body: account}),
		payCase("accounts get", []string{"a/%"}, nil, "", "2.0.0", exchange{method: "GET", path: "/accounts/a%2F%25"}),
		payCase("accounts list", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/accounts", query: "pageSize=100"}),
		payCase("accounts balances", []string{"a"}, map[string]string{"cursor": "next"}, "", "3.4.8", exchange{method: "GET", path: "/accounts/a/balances", query: "cursor=next"}),
		payCase("payments create", nil, confirm, payment, "3.4.8", exchange{method: "POST", path: "/payments", body: payment}),
		payCase("payments get", []string{"p"}, nil, "", "3.4.8", exchange{method: "GET", path: "/payments/p"}),
		payCase("payments list", nil, map[string]string{"page-size": "2"}, "", "2.5.0", exchange{method: "GET", path: "/payments", query: "pageSize=2"}),
		payCase("payments set-metadata", []string{"p", "key=value=extra"}, confirm, "", "3.4.8", exchange{method: "PATCH", path: "/payments/p/metadata", body: `{"key":"value=extra"}`}),
		payCase("bank-accounts create", nil, confirm, `{"name":"bank"}`, "3.4.8", exchange{method: "POST", path: "/v3/bank-accounts", body: `{"name":"bank"}`}),
		payCase("bank-accounts create", nil, confirm, `{"name":"bank","country":"FR"}`, "2.5.0", exchange{method: "POST", path: "/bank-accounts", body: `{"name":"bank","country":"FR"}`}),
		payCase("bank-accounts get", []string{"b"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/bank-accounts/b"}),
		payCase("bank-accounts get", []string{"b"}, nil, "", "2.5.0", exchange{method: "GET", path: "/bank-accounts/b"}),
		payCase("bank-accounts list", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/bank-accounts", query: "pageSize=100"}),
		payCase("bank-accounts forward", []string{"b", "conn"}, confirm, "", "3.4.8", exchange{method: "POST", path: "/v3/bank-accounts/b/forward", body: `{"connectorID":"conn"}`}),
		payCase("bank-accounts forward", []string{"b", "conn"}, confirm, "", "2.5.0", exchange{method: "POST", path: "/bank-accounts/b/forward", body: `{"connectorID":"conn"}`}),
		payCase("bank-accounts update-metadata", []string{"b"}, map[string]string{"confirm": "true", "metadata": "key=value"}, "", "3.4.8", exchange{method: "PATCH", path: "/v3/bank-accounts/b/metadata", body: `{"metadata":{"key":"value"}}`}),
		payCase("pools create", nil, confirm, `{"name":"pool","accountIDs":["a"]}`, "2.0.0", exchange{method: "POST", path: "/pools", body: `{"name":"pool","accountIDs":["a"]}`}),
		payCase("pools create", nil, confirm, `{"name":"pool","query":{"$match":{"amount":9007199254740993}}}`, "3.4.8", exchange{method: "POST", path: "/v3/pools", body: `{"name":"pool","query":{"$match":{"amount":9007199254740993}}}`}),
		payCase("pools get", []string{"pool"}, nil, "", "3.4.8", exchange{method: "GET", path: "/pools/pool"}),
		payCase("pools list", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/pools", query: "pageSize=100"}),
		payCase("pools delete", []string{"pool"}, confirm, "", "3.4.8", exchange{method: "DELETE", path: "/pools/pool", status: 204}),
		payCase("pools balances", []string{"pool", "2026-01-01T00:00:00Z"}, nil, "", "3.4.8", exchange{method: "GET", path: "/pools/pool/balances", query: "at=2026-01-01T00%3A00%3A00Z"}),
		payCase("pools latest-balances", []string{"pool"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/pools/pool/balances/latest"}),
		payCase("pools latest-balances", []string{"pool"}, nil, "", "2.0.0", exchange{method: "GET", path: "/pools/pool/balances/latest"}),
		payCase("pools add-account", []string{"pool", "a"}, nil, "", "3.4.8", exchange{method: "POST", path: "/pools/pool/accounts", body: `{"accountID":"a"}`}),
		payCase("pools remove-account", []string{"pool", "a"}, confirm, "", "3.4.8", exchange{method: "DELETE", path: "/pools/pool/accounts/a"}),
		payCase("pools update-query", []string{"pool"}, confirm, `{"query":{"$match":{"type":"INTERNAL"}}}`, "3.4.8", exchange{method: "PATCH", path: "/v3/pools/pool/query", body: `{"query":{"$match":{"type":"INTERNAL"}}}`}),
		payCase("transfer-initiation create", nil, confirm, initiation, "3.4.8", exchange{method: "POST", path: "/transfer-initiations", body: initiation}),
		payCase("transfer-initiation get", []string{"ti"}, nil, "", "3.4.8", exchange{method: "GET", path: "/transfer-initiations/ti"}),
		payCase("transfer-initiation list", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/transfer-initiations", query: "pageSize=100"}),
		payCase("transfer-initiation approve", []string{"ti"}, confirm, "", "3.4.8", exchange{method: "POST", path: "/v3/payment-initiations/ti/approve"}),
		payCase("transfer-initiation reject", []string{"ti"}, confirm, "", "3.4.8", exchange{method: "POST", path: "/v3/payment-initiations/ti/reject"}),
		payCase("transfer-initiation retry", []string{"ti"}, confirm, "", "3.4.8", exchange{method: "POST", path: "/transfer-initiations/ti/retry"}),
		payCase("transfer-initiation reverse", []string{"ti"}, confirm, reversal, "3.4.8", exchange{method: "POST", path: "/transfer-initiations/ti/reverse", body: reversal}),
		payCase("transfer-initiation delete", []string{"ti"}, confirm, "", "3.4.8", exchange{method: "DELETE", path: "/transfer-initiations/ti"}),
		payCase("transfer-initiation update-status", []string{"ti", "VALIDATED"}, confirm, "", "2.0.0", exchange{method: "POST", path: "/transfer-initiations/ti/status", body: `{"status":"VALIDATED"}`}),
		payCase("tasks get", []string{"task"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/tasks/task"}),
		payCase("orders get", []string{"order"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/orders/order"}),
		payCase("conversions get", []string{"conv"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/conversions/conv"}),
		payCase("orders list", nil, map[string]string{"connector-id": "c", "reference": "ref", "direction": "BUY", "status": "FILLED", "type": "MARKET", "source-asset": "USD/2", "destination-asset": "BTC/8", "page-size": "2"}, "", "3.4.8", exchange{method: "GET", path: "/v3/orders", query: "pageSize=2", body: `{"$and":[{"$match":{"connector_id":"c"}},{"$match":{"reference":"ref"}},{"$match":{"direction":"BUY"}},{"$match":{"status":"FILLED"}},{"$match":{"type":"MARKET"}},{"$match":{"source_asset":"USD/2"}},{"$match":{"destination_asset":"BTC/8"}}]}`}),
		payCase("conversions list", nil, map[string]string{"cursor": "next", "reference": "ignored-on-next-page"}, "", "3.4.8", exchange{method: "GET", path: "/v3/conversions", query: "cursor=next"}),
		payCase("conversions list", nil, map[string]string{"status": "COMPLETED"}, "", "3.4.8", exchange{method: "GET", path: "/v3/conversions", query: "pageSize=100", body: `{"$and":[{"$match":{"status":"COMPLETED"}}]}`}),
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}

func TestConnectorContracts(t *testing.T) {
	t.Parallel()
	configs := `{"data":{"Stripe":{},"GENERIC":{},"NewProvider":{}}}`
	cases := []commandCase{
		payCase("connectors install", []string{"dummypay"}, map[string]string{"confirm": "true"}, `{"name":"qa-hidden","directory":"/tmp"}`, "3.4.8",
			exchange{method: "GET", path: "/v3/connectors/configs", response: `{"data":{"stripe":{}}}`},
			exchange{method: "POST", path: "/v3/connectors/install/dummypay", body: `{"name":"qa-hidden","directory":"/tmp","provider":"dummypay"}`}),
		payCase("connectors list", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors", query: "pageSize=10"}),
		payCase("connectors list", nil, nil, "", "2.0.0", exchange{method: "GET", path: "/connectors", response: `{"data":[{"connectorID":"conn","provider":"STRIPE","name":"stripe"}]}`}),
		payCase("connectors list-available", nil, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors/configs"}),
		payCase("connectors list-available", nil, nil, "", "2.0.0", exchange{method: "GET", path: "/connectors/configs"}),
		payCase("connectors install", []string{"newprovider"}, map[string]string{"confirm": "true"}, `{"name":"test","provider":"stale","amount":9007199254740993}`, "3.4.8", exchange{method: "GET", path: "/v3/connectors/configs", response: configs}, exchange{method: "POST", path: "/v3/connectors/install/newprovider", body: `{"name":"test","provider":"NewProvider","amount":9007199254740993}`}),
		payCase("connectors install", []string{"bankingcircle"}, map[string]string{"confirm": "true"}, `{"name":"test"}`, "2.0.0", exchange{method: "POST", path: "/connectors/BANKING-CIRCLE", body: `{"name":"test"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn/%"}, `{"name":"test"}`, "3.4.8", exchange{method: "GET", path: "/v3/connectors/configs", response: configs}, exchange{method: "PATCH", path: "/v3/connectors/conn%2F%25/config", body: `{"name":"test","provider":"Stripe"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn"}, `{"name":"test"}`, "2.0.0", exchange{method: "POST", path: "/connectors/STRIPE/conn/config", body: `{"name":"test"}`}),
		payCase("connectors get-config", nil, map[string]string{"connector-id": "conn"}, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors/conn/config"}),
		payCase("connectors get-config", nil, map[string]string{"provider": "stripe", "connector-id": "conn"}, "", "2.0.0", exchange{method: "GET", path: "/connectors/STRIPE/conn/config"}),
		payCase("connectors get-config", nil, map[string]string{"provider": "stripe"}, "", "0.9.0", exchange{method: "GET", path: "/connectors/STRIPE/config"}),
		payCase("connectors get-config", nil, map[string]string{"connector-id": "conn"}, "", "2.0.0", exchange{method: "GET", path: "/connectors", response: `{"data":[{"connectorID":"conn","provider":"STRIPE"},{"connectorID":"other","provider":"WISE"}]}`}, exchange{method: "GET", path: "/connectors/STRIPE/conn/config"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "connector-id": "conn"}, "", "3.4.8", exchange{method: "DELETE", path: "/v3/connectors/conn"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "connector-id": "conn", "provider": "stripe"}, "", "2.0.0", exchange{method: "DELETE", path: "/connectors/STRIPE/conn"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "provider": "stripe"}, "", "0.9.0", exchange{method: "DELETE", path: "/connectors/STRIPE"}),
		payCase("connectors schedules list", []string{"conn"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors/conn/schedules", query: "pageSize=100"}),
		payCase("connectors schedules get", []string{"conn", "schedule"}, nil, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors/conn/schedules/schedule"}),
		payCase("connectors schedules instances list", []string{"conn", "schedule"}, map[string]string{"cursor": "next"}, "", "3.4.8", exchange{method: "GET", path: "/v3/connectors/conn/schedules/schedule/instances", query: "cursor=next"}),
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { t.Parallel(); runCase(t, tc) })
	}
}
