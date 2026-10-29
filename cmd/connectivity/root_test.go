package connectivity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandTreeMatchesConnectivityUX(t *testing.T) {
	cmd := NewCommand()
	require.Equal(t, "connectivity", cmd.Name())

	for _, path := range [][]string{
		{"status"},
		{"connectors", "list"},
		{"connectors", "show"},
		{"connections", "list"},
		{"connections", "show"},
		{"connections", "install"},
		{"connections", "configure"},
		{"connections", "uninstall"},
		{"connections", "suspend"},
		{"connections", "unsuspend"},
	} {
		found, _, err := cmd.Find(path)
		require.NoError(t, err)
		require.Equal(t, path[len(path)-1], found.Name())
	}

	old, _, _ := cmd.Find([]string{"connectorinstances"})
	require.NotEqual(t, "connectorinstances", old.Name())

	require.NotNil(t, cmd.PersistentFlags().Lookup("stack"))
	require.NotNil(t, cmd.PersistentFlags().Lookup("organization"))
}
