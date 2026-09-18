package connectivity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandTreeMatchesConnectivityUX(t *testing.T) {
	cmd := NewCommand()
	require.Equal(t, "connectivity", cmd.Name())

	for _, path := range [][]string{
		{"connectors", "list"},
		{"connectors", "show"},
		{"connectorinstances", "list"},
		{"connectorinstances", "show"},
		{"connectorinstances", "install"},
		{"connectorinstances", "configure"},
		{"connectorinstances", "uninstall"},
	} {
		found, _, err := cmd.Find(path)
		require.NoError(t, err)
		require.Equal(t, path[len(path)-1], found.Name())
	}

	require.NotNil(t, cmd.PersistentFlags().Lookup("stack"))
	require.NotNil(t, cmd.PersistentFlags().Lookup("organization"))
}
