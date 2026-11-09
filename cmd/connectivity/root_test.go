package connectivity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandTreeExposesServerInfosAndHidesTheConfigAliasGroup(t *testing.T) {
	root := NewCommand()

	serverInfos, _, err := root.Find([]string{"si"})
	require.NoError(t, err)
	require.Equal(t, "server-infos", serverInfos.Name())

	config, _, err := root.Find([]string{"connections", "config", "update"})
	require.NoError(t, err)
	require.True(t, config.Parent().Hidden, "config only duplicates configure")
	require.Equal(t, "update", config.Name(), "the hidden group keeps working")
}
