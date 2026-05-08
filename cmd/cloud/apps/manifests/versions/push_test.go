package versions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestPush_JSONOutput_LowercaseKeys(t *testing.T) {
	store := &Push{ManifestID: "mnf-abc", Version: 4}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "mnf-abc", data["manifestId"])
	require.EqualValues(t, 4, data["version"])

	for _, capitalized := range []string{"ManifestID", "Version"} {
		_, present := data[capitalized]
		require.False(t, present, "must not surface Go-cased key %q", capitalized)
	}
}
