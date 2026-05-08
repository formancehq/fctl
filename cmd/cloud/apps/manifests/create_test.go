package manifests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

// TestCreate_JSONOutput_LowercaseKeys verifies the Create store serializes
// to JSON with lowercase, idiomatic keys (`id`, `name`, `version`) under
// `data`. Without explicit json tags Go marshals exported fields as-is
// (Go-cased), which would produce `{"data":{"ID":..., "Name":..., "Version":...}}`
// and break any consumer (script, dashboard) reading `.data.id`. The
// provision-example-apps.sh script tripped on this exact bug.
func TestCreate_JSONOutput_LowercaseKeys(t *testing.T) {
	store := &Create{
		ID:      "mnf-abc123",
		Name:    "my-manifest",
		Version: 1,
	}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok, "data should be an object")

	require.Equal(t, "mnf-abc123", data["id"], "expected lowercase 'id' key")
	require.Equal(t, "my-manifest", data["name"], "expected lowercase 'name' key")
	require.EqualValues(t, 1, data["version"], "expected lowercase 'version' key")

	// Negative: capital aliases must NOT exist (would indicate missing tags).
	for _, capitalized := range []string{"ID", "Name", "Version"} {
		_, ok := data[capitalized]
		require.False(t, ok, "JSON output should not contain Go-cased key %q", capitalized)
	}
}
