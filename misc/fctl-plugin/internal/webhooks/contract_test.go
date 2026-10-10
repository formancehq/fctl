package webhooks

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestWebhooksContracts(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	cases := []testutil.Case{
		{Name: "config filters", Command: "webhooks list", Flags: map[string]string{"config-id": "config/a", "endpoint": "https://example.org/hook"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/configs", Query: "id=config%2Fa&endpoint=https%3A%2F%2Fexample.org%2Fhook"}}},
		{Name: "create", Command: "webhooks create", Args: []string{"https://example.org/hook", "ledger.transaction.created", "wallets.created"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/configs", Body: `{"endpoint":"https://example.org/hook","eventTypes":["ledger.transaction.created","wallets.created"],"secret":""}`}}},
		{Name: "update omit secret", Command: "webhooks update", Args: []string{"id/%", "https://example.org/new", "event"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/configs/id%2F%25", Body: `{"endpoint":"https://example.org/new","eventTypes":["event"]}`}}},
		{Name: "update explicit empty secret", Command: "webhooks update", Args: []string{"id", "https://example.org/new", "event"}, Flags: map[string]string{"confirm": "true", "secret": ""}, Changed: map[string]bool{"secret": true}, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/configs/id", Body: `{"endpoint":"https://example.org/new","eventTypes":["event"],"secret":""}`}}},
		{Name: "rotate secret", Command: "webhooks change-secret", Args: []string{"id"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/configs/id/secret/change", Body: `{"secret":""}`}}},
		{Name: "activate", Command: "webhooks activate", Args: []string{"id"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/configs/id/activate"}}},
		{Name: "deactivate", Command: "webhooks deactivate", Args: []string{"id"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/configs/id/deactivate"}}},
		{Name: "delete", Command: "webhooks delete", Args: []string{"id"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "DELETE", Path: "/configs/id", Status: 204}}},
		{Name: "send test", Command: "webhooks test", Args: []string{"id"}, Flags: confirm, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/configs/id/test"}}},
		{Name: "delivery filters", Command: "webhooks deliveries list", Flags: map[string]string{"status": "failed", "config-id": "cfg", "created-at-from": "2026-01-01T00:00:00Z", "created-at-to": "2026-02-01T00:00:00Z", "page-size": "20", "cursor": "opaque +/="}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/deliveries", Query: "status=failed&configId=cfg&createdAtFrom=2026-01-01T00%3A00%3A00Z&createdAtTo=2026-02-01T00%3A00%3A00Z&pageSize=20&cursor=opaque+%2B%2F%3D"}}},
		{Name: "delivery show", Command: "webhooks deliveries show", Args: []string{"delivery/a"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/deliveries/delivery%2Fa"}}},
		{Name: "attempts", Command: "webhooks deliveries attempts", Args: []string{"delivery"}, Flags: map[string]string{"cursor": "next", "page-size": "2"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/deliveries/delivery/attempts", Query: "cursor=next&pageSize=2"}}},
		{Name: "replay", Command: "webhooks deliveries replay", Args: []string{"delivery"}, Flags: map[string]string{"confirm": "true", "idempotency-key": "replay-1"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/deliveries/delivery/replay", Key: "replay-1"}}},
		{Name: "replay page", Command: "webhooks deliveries replay-bulk", Flags: map[string]string{"confirm": "true", "idempotency-key": "bulk-1", "created-at-from": "2026-01-01T00:00:00Z", "created-at-to": "2026-02-01T00:00:00Z", "status": "FAILED,pending", "config-id": "[\"cfg/1\",\"cfg2\"]", "cursor": "next", "page-size": "2"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/deliveries/replay", Key: "bulk-1", Body: `{"createdAtFrom":"2026-01-01T00:00:00Z","createdAtTo":"2026-02-01T00:00:00Z","statuses":["failed","pending"],"configIds":["cfg/1","cfg2"],"cursor":"next","pageSize":2}`}}},
		{Name: "replay defaults", Command: "webhooks deliveries replay-bulk", Flags: map[string]string{"confirm": "true", "idempotency-key": "bulk-2", "created-at-from": "2026-01-01T00:00:00Z"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/deliveries/replay", Key: "bulk-2", Body: `{"createdAtFrom":"2026-01-01T00:00:00Z","statuses":["failed","pending"],"pageSize":100}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
