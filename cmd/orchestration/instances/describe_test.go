package instances

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/orchestration"
)

func TestStageNamesFromWireResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"wallet", `{"wallet":{"id":"wallet-1","balance":"main"}}`, "wallet 'wallet-1' (balance: main)"},
		{"account", `{"account":{"id":"customer:1","ledger":"default"}}`, "account 'customer:1' (ledger: default)"},
		{"payment", `{"payment":{"id":"payment-1"}}`, "payment 'payment-1'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source orchestration.StageSendSource
			require.NoError(t, json.Unmarshal([]byte(tc.body), &source))
			require.Equal(t, tc.want, stageSourceName(&source))
		})
	}

	var destination orchestration.StageSendDestination
	require.NoError(t, json.Unmarshal([]byte(`{"payment":{"psp":"stripe"}}`), &destination))
	require.Equal(t, "stripe", stageDestinationName(&destination))
}
