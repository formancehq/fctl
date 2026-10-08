package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCommandRegistersConnectivity(t *testing.T) {
	found, _, err := NewRootCommand().Find([]string{"connectivity"})
	require.NoError(t, err)
	require.Equal(t, "connectivity", found.Name())
}
