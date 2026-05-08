package apps

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestBindManifest_JSONOutput_LowercaseKeys(t *testing.T) {
	store := &BindManifest{
		AppID:      "app-abc",
		ManifestID: "mnf-xyz",
	}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "app-abc", data["appId"])
	require.Equal(t, "mnf-xyz", data["manifestId"])

	for _, capitalized := range []string{"AppID", "ManifestID"} {
		_, present := data[capitalized]
		require.False(t, present, "must not surface Go-cased key %q", capitalized)
	}
}

func TestUnbindManifest_JSONOutput_LowercaseKeys(t *testing.T) {
	store := &UnbindManifest{AppID: "app-abc"}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "app-abc", data["appId"])
	_, capitalized := data["AppID"]
	require.False(t, capitalized)
}
