package wallets

import (
	"maps"
	"reflect"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestWalletNameValidationMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "negative credit by name", Command: "wallets credit", Args: []string{"-1", "USD/2"}, Flags: map[string]string{"name": "Alice", "confirm": "true"}},
		{Name: "invalid source by name", Command: "wallets credit", Args: []string{"1", "USD/2"}, Flags: map[string]string{"name": "Alice", "source": "wallet=unknown:id", "confirm": "true"}},
		{Name: "invalid destination by name", Command: "wallets debit", Args: []string{"1", "USD/2"}, Flags: map[string]string{"name": "Alice", "destination": "account=", "confirm": "true"}},
		{Name: "invalid balance route by name", Command: "wallets balances show", Args: []string{".."}, Flags: map[string]string{"name": "Alice"}},
		{Name: "invalid expiry by name", Command: "wallets balances create", Args: []string{"promo"}, Flags: map[string]string{"name": "Alice", "expires-at": "tomorrow", "confirm": "true"}},
		{Name: "missing wallet", Command: "wallets show"},
		{Name: "dot wallet", Command: "wallets show", Flags: map[string]string{"id": ".."}},
		{Name: "conflicting target", Command: "wallets show", Flags: map[string]string{"id": "w", "name": "Alice"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFailure(t, New, tc, "")
		})
	}
}

func TestWalletRouteValidationDoesNotMutateFlags(t *testing.T) {
	t.Parallel()
	op := targetWallet(commandapi.Leaf("show", "show", "GET", "wallets/@id", 0, 0), true)
	request := pluginsdk.ExecuteRequest{Flags: map[string]string{"name": "Alice"}}
	original := maps.Clone(request.Flags)
	if err := op.ValidateRoute(request, op); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request.Flags, original) {
		t.Fatalf("route validation changed the target flags: %v", request.Flags)
	}
	if _, err := commandapi.Route(op.Segments, request); err == nil {
		t.Fatal("generic route accepted an unresolved wallet name")
	}
}

func TestWalletResolvedIDsAreValidated(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "invalid resolved ID", Command: "wallets show", Flags: map[string]string{"name": "Alice"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=Alice&pageSize=2", Response: `{"cursor":{"data":[{"id":".."}]}}`}}},
		{Name: "ambiguous next page", Command: "wallets show", Flags: map[string]string{"name": "Alice"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=Alice&pageSize=2", Response: `{"cursor":{"data":[{"id":"w"}],"hasMore":true}}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFailure(t, New, tc, "")
		})
	}
}
