package payments

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "empty account payload", Command: "payments accounts create", Body: `{}`, Flags: map[string]string{"confirm": "true"}},
		{Name: "duplicate payment metadata", Command: "payments payments set-metadata", Args: []string{"id", "x=1", "x=2"}, Flags: map[string]string{"confirm": "true"}},
		{Name: "missing payment metadata", Command: "payments payments set-metadata", Args: []string{"id"}, Flags: map[string]string{"confirm": "true"}},
		{Name: "bad transfer status", Command: "payments transfer-initiation update-status", Args: []string{"id", "other"}, Flags: map[string]string{"confirm": "true"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunFailure(t, New, tc, "") })
	}
}

func TestDependentReadFailuresPreventMutation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tc   testutil.Case
		want string
	}{
		{testutil.Case{Command: "payments connectors install", Args: []string{"DUMMYPAY"}, Body: `{"name":"qa"}`, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/_info", Response: `{"version":"3.4.8"}`}, {Method: "GET", Path: "/v3/connectors/configs", Response: `{"data":[]}`}}}, "missing a data object"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) { t.Parallel(); testutil.RunFailure(t, New, tc.tc, tc.want) })
	}
}

func TestPaymentsVersionAndSchemaFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tc            testutil.Case
		version, want string
	}{
		{testutil.Case{Command: "payments orders list"}, "3.2.9", "3.3.0"},
		{testutil.Case{Command: "payments pools update-query", Args: []string{"id"}, Body: `{"query":{}}`, Flags: map[string]string{"confirm": "true"}}, "3.0.0", "3.1.0"},
		{testutil.Case{Command: "payments tasks get", Args: []string{"id"}}, "2.4.0", "3.0.0"},
		{testutil.Case{Command: "payments transfer-initiation update-status", Args: []string{"id", "VALIDATED"}, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "unavailable"},
		{testutil.Case{Command: "payments payments list"}, "4.0.0", "unsupported"},
		{testutil.Case{Command: "payments payments list"}, "development", "unrecognized"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","query":{}}`, Flags: map[string]string{"confirm": "true"}}, "2.0.0", "dynamic pools"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","accountIDs":[""]}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "identifier"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","accountIDs":1}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "array"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","query":[]}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "query"},
		{testutil.Case{Command: "payments bank-accounts create", Body: `{"name":"bank"}`, Flags: map[string]string{"confirm": "true"}}, "2.4.0", "country"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":1}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "string"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","metadata":{"n":1}}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "metadata"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","validated":"true"}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "boolean"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","amount":-1}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "nonnegative"},
		{testutil.Case{Command: "payments pools create", Body: `{"name":"pool","createdAt":5}`, Flags: map[string]string{"confirm": "true"}}, "3.4.8", "RFC3339"},
	}
	for _, tc := range cases {
		t.Run(tc.tc.Command+" "+tc.version+" "+tc.want, func(t *testing.T) {
			t.Parallel()
			tc.tc.Exchanges = []testutil.Exchange{{Method: "GET", Path: "/_info", Response: `{"version":"` + tc.version + `"}`}}
			testutil.RunFailure(t, New, tc.tc, tc.want)
		})
	}
}

func TestHistoricalUnderscoreAliases(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Command: "payments bank_accounts get", Args: []string{"id"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/_info", Response: `{"version":"3.4.8"}`}, {Method: "GET", Path: "/v3/bank-accounts/id"}}},
		{Command: "payments transfer_initiation update_status", Args: []string{"id", "VALIDATED"}, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/_info", Response: `{"version":"2.4.0"}`}, {Method: "POST", Path: "/transfer-initiations/id/status", Body: `{"status":"VALIDATED"}`}}},
		{Command: "payments payment_initiations approve", Args: []string{"id"}, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/_info", Response: `{"version":"3.4.8"}`}, {Method: "POST", Path: "/v3/payment-initiations/id/approve"}}},
	}
	for _, tc := range cases {
		t.Run(tc.Command, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
