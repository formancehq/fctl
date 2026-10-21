package payments

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestConnectorContracts(t *testing.T) {
	t.Parallel()
	configs := `{"data":{"Stripe":{},"GENERIC":{},"NewProvider":{}}}`
	cases := []testutil.Case{
		payCase("connectors install", []string{"dummypay"}, map[string]string{"confirm": "true"}, `{"name":"qa-hidden","directory":"/tmp"}`, "3.4.8",
			testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs", Response: `{"data":{"stripe":{}}}`},
			testutil.Exchange{Method: "POST", Path: "/v3/connectors/install/dummypay", Body: `{"name":"qa-hidden","directory":"/tmp","provider":"dummypay"}`}),
		payCase("connectors list", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors", Query: "pageSize=10"}),
		payCase("connectors list", nil, nil, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/connectors", Response: `{"data":[{"connectorID":"conn","provider":"STRIPE","name":"stripe"}]}`}),
		payCase("connectors list-available", nil, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs"}),
		payCase("connectors list-available", nil, nil, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/connectors/configs"}),
		payCase("connectors install", []string{"newprovider"}, map[string]string{"confirm": "true"}, `{"name":"test","provider":"stale","amount":9007199254740993}`, "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs", Response: configs}, testutil.Exchange{Method: "POST", Path: "/v3/connectors/install/newprovider", Body: `{"name":"test","provider":"NewProvider","amount":9007199254740993}`}),
		payCase("connectors install", []string{"bankingcircle"}, map[string]string{"confirm": "true"}, `{"name":"test"}`, "2.0.0", testutil.Exchange{Method: "POST", Path: "/connectors/BANKING-CIRCLE", Body: `{"name":"test"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn/%"}, `{"name":"test"}`, "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/configs", Response: configs}, testutil.Exchange{Method: "PATCH", Path: "/v3/connectors/conn%2F%25/config", Body: `{"name":"test","provider":"Stripe"}`}),
		payCase("connectors update-config", []string{"stripe"}, map[string]string{"confirm": "true", "connector-id": "conn"}, `{"name":"test"}`, "2.0.0", testutil.Exchange{Method: "POST", Path: "/connectors/STRIPE/conn/config", Body: `{"name":"test"}`}),
		payCase("connectors get-config", nil, map[string]string{"connector-id": "conn"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/conn/config"}),
		payCase("connectors get-config", nil, map[string]string{"provider": "stripe", "connector-id": "conn"}, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/connectors/STRIPE/conn/config"}),
		payCase("connectors get-config", nil, map[string]string{"provider": "stripe"}, "", "0.9.0", testutil.Exchange{Method: "GET", Path: "/connectors/STRIPE/config"}),
		payCase("connectors get-config", nil, map[string]string{"connector-id": "conn"}, "", "2.0.0", testutil.Exchange{Method: "GET", Path: "/connectors", Response: `{"data":[{"connectorID":"conn","provider":"STRIPE"},{"connectorID":"other","provider":"WISE"}]}`}, testutil.Exchange{Method: "GET", Path: "/connectors/STRIPE/conn/config"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "connector-id": "conn"}, "", "3.4.8", testutil.Exchange{Method: "DELETE", Path: "/v3/connectors/conn"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "connector-id": "conn", "provider": "stripe"}, "", "2.0.0", testutil.Exchange{Method: "DELETE", Path: "/connectors/STRIPE/conn"}),
		payCase("connectors uninstall", nil, map[string]string{"confirm": "true", "provider": "stripe"}, "", "0.9.0", testutil.Exchange{Method: "DELETE", Path: "/connectors/STRIPE"}),
		payCase("connectors schedules list", []string{"conn"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/conn/schedules", Query: "pageSize=100"}),
		payCase("connectors schedules get", []string{"conn", "schedule"}, nil, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/conn/schedules/schedule"}),
		payCase("connectors schedules instances list", []string{"conn", "schedule"}, map[string]string{"cursor": "next"}, "", "3.4.8", testutil.Exchange{Method: "GET", Path: "/v3/connectors/conn/schedules/schedule/instances", Query: "cursor=next"}),
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
