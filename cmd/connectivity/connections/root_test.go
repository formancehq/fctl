package connections

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/cmd/connectivity/internal/client"
)

func TestConfigurationTreePreservesUpdateAndLegacyRoutes(t *testing.T) {
	for _, path := range [][]string{
		{"config", "update"}, {"configuration", "update"}, {"configure"}, {"update"}, {"c"},
	} {
		t.Run(path[0]+"/"+path[len(path)-1], func(t *testing.T) {
			current := instanceFixture("update")
			var gotPatch connectivityclient.ConnectionPatch
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(_ context.Context, name string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					require.Equal(t, "update", name)
					gotPatch = patch
					return &current, nil
				},
			}
			command := NewCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
			args := append(append([]string{}, path...), "update", "--poll-interval=15s", "--confirm")
			output, err := executeCommand(command, args...)
			require.NoError(t, err)
			require.Equal(t, connectivityclient.ConnectionPatch{"configuration": map[string]any{"pollInterval": "15s"}}, gotPatch)
			require.Contains(t, output, `Connection "update" configured.`)
		})
	}
}

func TestConfigurationGroupShowsHelpWithoutAuthenticating(t *testing.T) {
	command := NewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	output, err := executeCommand(command, "config")
	require.NoError(t, err)
	require.Contains(t, output, "update")
}
