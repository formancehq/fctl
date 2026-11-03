package bankaccounts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/payments"
)

func TestV1BankAccountCursorPreservesCLIOutput(t *testing.T) {
	const fixture = `{"cursor":{"pageSize":1,"hasMore":true,"next":"opaque-token","data":[{"id":"bank-1","name":"Treasury","country":"FR","createdAt":"2026-01-02T03:04:05Z","metadata":{"reference":"invoice-1"},"relatedAccounts":[{"accountID":"account-1","createdAt":"2026-01-02T03:04:05Z"}]}]}}`
	var response payments.BankAccountsCursor
	require.NoError(t, json.Unmarshal([]byte(fixture), &response))

	store := ListStore{Cursor: ToV3BankAccountCursor(&response.Cursor)}
	encoded, err := json.Marshal(store)
	require.NoError(t, err)
	require.JSONEq(t, fixture, string(encoded))
}
