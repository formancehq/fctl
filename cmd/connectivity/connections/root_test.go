package connections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigurationGroupShowsHelpWithoutAuthenticating(t *testing.T) {
	command := testNewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	output, err := executeCommand(command, "config")
	require.NoError(t, err)
	require.Contains(t, output, "update")
}
