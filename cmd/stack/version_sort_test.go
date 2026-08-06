package stack

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

func TestSortRegionVersionsByLatest(t *testing.T) {
	versions := []components.Version{
		{Name: "v1.2.0"},
		{Name: "1.10.0"},
		{Name: "v2.0.0-rc.1"},
		{Name: "v2.0.0"},
		{Name: "v1.9.0"},
	}

	sorted := sortRegionVersionsByLatest(versions)

	require.Equal(t, []string{
		"v2.0.0",
		"v2.0.0-rc.1",
		"1.10.0",
		"v1.9.0",
		"v1.2.0",
	}, []string{
		sorted[0].GetName(),
		sorted[1].GetName(),
		sorted[2].GetName(),
		sorted[3].GetName(),
		sorted[4].GetName(),
	})
	require.Equal(t, "v1.2.0", versions[0].GetName())
}

func TestSortVersionNamesByLatest(t *testing.T) {
	require.Equal(t,
		[]string{"v1.11.0", "1.10.0", "v1.2.0"},
		sortVersionNamesByLatest([]string{"v1.2.0", "v1.11.0", "1.10.0"}),
	)
}

func TestSortVersionNamesByLatestWithShortVersions(t *testing.T) {
	require.Equal(t,
		[]string{"v3.2-rc", "v3.1", "v3.0", "v2.2", "v2.1"},
		sortVersionNamesByLatest([]string{"v3.0", "v2.2", "v3.1", "v3.2-rc", "v2.1"}),
	)
}

func TestIsVersionNewerThanCurrent(t *testing.T) {
	require.True(t, isVersionNewerThanCurrent("1.10.0", "v1.9.0"))
	require.False(t, isVersionNewerThanCurrent("v1.9.0", "1.10.0"))
	require.True(t, isVersionNewerThanCurrent("v3.2-rc", "v3.1"))
}
