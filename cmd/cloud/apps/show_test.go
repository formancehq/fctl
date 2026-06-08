package apps

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestShow_JSONOutput_PreservesSDKShape(t *testing.T) {
	stackID := "stk-abc"
	deployedManifestID := "mnf-old"
	deployedVersion := int64(1)

	store := &Show{
		App: components.App{
			ID:      "app-abc",
			Name:    "my-app",
			StackID: &stackID,
			CurrentManifest: &components.AppCurrentManifest{
				ID:            "mnf-new",
				Name:          "my-manifest",
				LatestVersion: 3,
				Divergence: &components.AppManifestDivergence{
					Kind:               components.KindRebound,
					DeployedManifestID: &deployedManifestID,
					DeployedVersion:    &deployedVersion,
					LatestVersion:      3,
				},
			},
		},
	}

	out, err := json.Marshal(fctl.ExportedData{Data: store})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(out, &raw))

	data, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "app-abc", data["id"])
	require.Equal(t, "stk-abc", data["stackId"])

	cm, ok := data["currentManifest"].(map[string]any)
	require.True(t, ok, "currentManifest must be present and named lowercase")
	require.Equal(t, "mnf-new", cm["id"])
	require.EqualValues(t, 3, cm["latestVersion"])

	div, ok := cm["divergence"].(map[string]any)
	require.True(t, ok, "divergence must be present")
	require.Equal(t, "rebound", div["kind"])
	require.Equal(t, "mnf-old", div["deployedManifestId"])
	require.EqualValues(t, 1, div["deployedVersion"])
}

func TestShow_PlainRender_IncludesDivergenceSection(t *testing.T) {
	deployedManifestID := "mnf-old"
	deployedVersion := int64(1)

	c := NewShowCtrl()
	c.store.App = components.App{
		ID:   "app-abc",
		Name: "my-app",
		CurrentManifest: &components.AppCurrentManifest{
			ID:            "mnf-new",
			Name:          "my-manifest",
			LatestVersion: 3,
			Divergence: &components.AppManifestDivergence{
				Kind:               components.KindBehind,
				DeployedManifestID: &deployedManifestID,
				DeployedVersion:    &deployedVersion,
				LatestVersion:      3,
			},
		},
	}

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	require.NoError(t, c.Render(cmd, nil))

	// pterm.DefaultSection writes to os.Stdout directly; only the bullet
	// list goes through cmd.Out. So we assert on bullet content, not the
	// section header.
	got := stripANSI(buf.String())
	require.Contains(t, got, "ID: mnf-new", "should print bound manifest id")
	require.Contains(t, got, "Name: my-manifest", "should print bound manifest name")
	require.Contains(t, got, "Latest Version: 3", "should print latest version of bound manifest")
	require.Contains(t, got, "behind", "should print divergence kind")
	require.Contains(t, got, "Deployed manifest: mnf-old")
	require.Contains(t, got, "Deployed version: 1")
	require.Contains(t, got, "Latest version: 3")
}

func TestShow_PlainRender_OmitsDivergenceWhenAbsent(t *testing.T) {
	c := NewShowCtrl()
	c.store.App = components.App{ID: "app-abc", Name: "my-app"}

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	require.NoError(t, c.Render(cmd, nil))

	got := stripANSI(buf.String())
	require.NotContains(t, got, "rebound")
	require.NotContains(t, got, "Deployed manifest")
}

func TestDivergenceHint_CoversAllKinds(t *testing.T) {
	for _, k := range []components.Kind{
		components.KindUndeployed,
		components.KindSynced,
		components.KindBehind,
		components.KindRebound,
	} {
		require.NotEmpty(t, divergenceHint(k), "missing hint for kind %q", k)
	}
	require.Empty(t, divergenceHint(components.Kind("unknown")))
}

// stripANSI removes pterm color/escape sequences so assertions can match on
// the raw text content without coupling to terminal formatting.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || r == 'K' || r == 'A' || r == 'B' || r == 'C' || r == 'D'):
			inEsc = false
		case inEsc:
			// swallow CSI body
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
