package manifests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestDelete_JSONOutput_LowercaseKeys(t *testing.T) {
	store := &Delete{ID: "mnf-abc123"}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "mnf-abc123", data["id"])
	_, capitalized := data["ID"]
	require.False(t, capitalized)
}
