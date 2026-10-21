package webhooks

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "delete confirmation", Command: "webhooks delete", Args: []string{"id"}},
		{Name: "route traversal", Command: "webhooks delete", Args: []string{".."}, Flags: map[string]string{"confirm": "true"}},
		{Name: "webhook userinfo", Command: "webhooks create", Args: []string{"https://user@example.invalid/hook", "event"}, Flags: map[string]string{"confirm": "true"}},
		{Name: "webhook scheme", Command: "webhooks create", Args: []string{"ftp://example.invalid/hook", "event"}, Flags: map[string]string{"confirm": "true"}},
		{Name: "empty event", Command: "webhooks create", Args: []string{"https://example.invalid/hook", " "}, Flags: map[string]string{"confirm": "true"}},
		{Name: "invalid secret", Command: "webhooks change-secret", Args: []string{"id", "not-base64"}, Flags: map[string]string{"confirm": "true"}},
		{Name: "reverse replay range", Command: "webhooks deliveries replay-bulk", Flags: map[string]string{"confirm": "true", "idempotency-key": "k", "created-at-from": "2026-02-01T00:00:00Z", "created-at-to": "2026-01-01T00:00:00Z"}},
		{Name: "missing replay start", Command: "webhooks deliveries replay-bulk", Flags: map[string]string{"confirm": "true", "idempotency-key": "k"}},
		{Name: "invalid replay status", Command: "webhooks deliveries replay-bulk", Flags: map[string]string{"confirm": "true", "idempotency-key": "k", "created-at-from": "2026-01-01T00:00:00Z", "status": "succeeded"}},
		{Name: "invalid delivery status", Command: "webhooks deliveries list", Flags: map[string]string{"status": "other"}},
		{Name: "invalid delivery timestamp", Command: "webhooks deliveries list", Flags: map[string]string{"created-at-from": "yesterday"}},
		{Name: "zero page", Command: "webhooks deliveries list", Flags: map[string]string{"page-size": "0"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFailure(t, New, tc, "")
		})
	}
}
