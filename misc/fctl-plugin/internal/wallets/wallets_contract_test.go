package wallets

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestWalletContracts(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "create", Command: "wallets create", Args: []string{"Alice"}, Flags: map[string]string{"confirm": "true", "metadata": "customer=42", "ik": "create-1"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/wallets", Key: "create-1", Body: `{"name":"Alice","metadata":{"customer":"42"}}`}}},
		{Name: "update", Command: "wallets update", Args: []string{"w/1"}, Flags: map[string]string{"confirm": "true", "metadata": "{\"customer\":\"43\"}", "ik": "update-1"}, Exchanges: []testutil.Exchange{{Method: "PATCH", Path: "/wallets/w%2F1", Key: "update-1", Body: `{"metadata":{"customer":"43"}}`}}},
		{Name: "metadata list", Command: "wallets list", Flags: map[string]string{"name": "Alice", "metadata": "customer=42", "page-size": "2"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=Alice&metadata%5Bcustomer%5D=42&pageSize=2"}}},
		{Name: "cursor list", Command: "wallets list", Flags: map[string]string{"cursor": "opaque +/=", "metadata": "customer=42"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "cursor=opaque+%2B%2F%3D"}}},
		{Name: "show id", Command: "wallets show", Flags: map[string]string{"id": "w/1"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets/w%2F1"}}},
		{Name: "show name", Command: "wallets show", Flags: map[string]string{"name": "Alice"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=Alice&pageSize=2", Response: `{"cursor":{"data":[{"id":"w/1"}],"hasMore":false}}`}, {Method: "GET", Path: "/wallets/w%2F1"}}},
		{Name: "create balance", Command: "wallets balances create", Args: []string{"promo"}, Flags: map[string]string{"confirm": "true", "id": "w", "expires-at": "2027-01-01T00:00:00Z", "priority": "-9007199254740993"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/wallets/w/balances", Body: `{"name":"promo","expiresAt":"2027-01-01T00:00:00Z","priority":-9007199254740993}`}}},
		{Name: "list balances", Command: "wallets balances list", Flags: map[string]string{"id": "w", "page-size": "2"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets/w/balances", Query: "pageSize=2"}}},
		{Name: "show balance", Command: "wallets balances show", Args: []string{"promo/1"}, Flags: map[string]string{"id": "w"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets/w/balances/promo%2F1"}}},
		{Name: "credit exact integer and subjects", Command: "wallets credit", Args: []string{"9007199254740993", "USD/2"}, Flags: map[string]string{"confirm": "true", "id": "w", "source": "account=world,wallet=name:Bob/promo", "balance": "main", "metadata": "reference=credit-1", "ik": "credit-1"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/wallets", Query: "name=Bob&pageSize=2", Response: `{"cursor":{"data":[{"id":"bob"}],"hasMore":false}}`}, {Method: "POST", Path: "/wallets/w/credit", Key: "credit-1", Body: `{"amount":{"amount":9007199254740993,"asset":"USD/2"},"balance":"main","metadata":{"reference":"credit-1"},"sources":[{"type":"ACCOUNT","identifier":"world"},{"type":"WALLET","identifier":"bob","balance":"promo"}]}`}}},
		{Name: "debit pending", Command: "wallets debit", Args: []string{"100", "USD/2"}, Flags: map[string]string{"confirm": "true", "id": "w", "pending": "true", "balance": "main,promo", "destination": "wallet=id:target", "description": "pending debit", "ik": "debit-1"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/wallets/w/debit", Key: "debit-1", Body: `{"amount":{"amount":100,"asset":"USD/2"},"pending":true,"balances":["main","promo"],"metadata":{},"description":"pending debit","destination":{"type":"WALLET","identifier":"target","balance":"main"}}`}}},
		{Name: "holds list", Command: "wallets holds list", Flags: map[string]string{"id": "w", "metadata": "customer=42"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/holds", Query: "walletID=w&metadata%5Bcustomer%5D=42&pageSize=100"}}},
		{Name: "hold show", Command: "wallets holds show", Args: []string{"h"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/holds/h"}}},
		{Name: "confirm hold", Command: "wallets holds confirm", Args: []string{"h"}, Flags: map[string]string{"confirm": "true", "amount": "9007199254740993", "final": "true", "ik": "confirm-1"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/holds/h/confirm", Key: "confirm-1", Body: `{"amount":9007199254740993,"final":true}`}}},
		{Name: "void hold", Command: "wallets holds void", Args: []string{"h"}, Flags: map[string]string{"confirm": "true", "ik": "void-1"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/holds/h/void", Key: "void-1"}}},
		{Name: "transactions list", Command: "wallets transactions list", Flags: map[string]string{"id": "w"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/transactions", Query: "walletID=w&pageSize=100"}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
