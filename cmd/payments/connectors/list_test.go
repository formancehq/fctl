package connectors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/payments"
)

func TestV1ConnectorResponsePreservesProvider(t *testing.T) {
	var response payments.ConnectorsResponse
	require.NoError(t, json.Unmarshal([]byte(`{"data":[{"provider":"STRIPE","connectorID":"connector-1","name":"Card payments","enabled":true}]}`), &response))
	require.Len(t, response.Data, 1)
	require.Equal(t, ConnectorData{
		ID: "connector-1", Name: "Card payments", Provider: "STRIPE",
	}, V1toConnectorData(response.Data[0]))
}
