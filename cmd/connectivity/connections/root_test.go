package connections

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func TestConfigurationTreePreservesUpdateAndLegacyRoutes(t *testing.T) {
	for _, path := range [][]string{
		{"config", "update"}, {"configuration", "update"}, {"configure"}, {"update"}, {"c"},
	} {
		t.Run(path[0]+"/"+path[len(path)-1], func(t *testing.T) {
			current := instanceFixture("update")
			var gotPatch map[string]any
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
				patch: func(_ context.Context, name string, patch map[string]any) (*connectivity.Connection, error) {
					require.Equal(t, "update", name)
					gotPatch = patch
					return &current, nil
				},
			}
			command := testNewCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
			args := append(append([]string{}, path...), "update", "--poll-interval=15s", "--confirm")
			output, err := executeCommand(command, args...)
			require.NoError(t, err)
			require.Equal(t, map[string]any{"configuration": map[string]any{"pollInterval": "15s"}}, gotPatch)
			require.Contains(t, output, `Connection "update" configured.`)
		})
	}
}

func TestConfigurationGroupShowsHelpWithoutAuthenticating(t *testing.T) {
	command := testNewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	output, err := executeCommand(command, "config")
	require.NoError(t, err)
	require.Contains(t, output, "update")
}
