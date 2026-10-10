package wallets

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "unexpected body", Command: "wallets show", Flags: map[string]string{"id": "id"}, Body: `{}`},
		{Name: "negative credit", Command: "wallets credit", Args: []string{"-1", "USD/2"}, Flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{Name: "fractional debit", Command: "wallets debit", Args: []string{"1.5", "USD/2"}, Flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{Name: "empty wallet asset", Command: "wallets debit", Args: []string{"1", " "}, Flags: map[string]string{"id": "wallet", "confirm": "true"}},
		{Name: "invalid source", Command: "wallets credit", Args: []string{"1", "USD/2"}, Flags: map[string]string{"id": "wallet", "source": "wallet=unknown:id", "confirm": "true"}},
		{Name: "invalid destination", Command: "wallets debit", Args: []string{"1", "USD/2"}, Flags: map[string]string{"id": "wallet", "destination": "account=", "confirm": "true"}},
		{Name: "bad metadata", Command: "wallets create", Args: []string{"wallet"}, Flags: map[string]string{"metadata": `{"amount":1}`, "confirm": "true"}},
		{Name: "expired timestamp format", Command: "wallets balances create", Args: []string{"balance"}, Flags: map[string]string{"id": "wallet", "expires-at": "tomorrow", "confirm": "true"}},
		{Name: "negative hold amount", Command: "wallets holds confirm", Args: []string{"hold"}, Flags: map[string]string{"amount": "-1", "confirm": "true"}},
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
		{testutil.Case{Command: "wallets credit", Args: []string{"1", "USD/2"}, Flags: map[string]string{"name": "missing", "confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=missing&pageSize=2", Response: `{"cursor":{"data":[]}}`}}}, "not found"},
		{testutil.Case{Command: "wallets show", Flags: map[string]string{"name": "duplicate"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=duplicate&pageSize=2", Response: `{"cursor":{"data":[{"id":"1"},{"id":"2"}]}}`}}}, "ambiguous"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) { t.Parallel(); testutil.RunFailure(t, New, tc.tc, tc.want) })
	}
}
