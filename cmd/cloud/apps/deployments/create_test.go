package deployments

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewCreate_RequiredFlags asserts the cobra command exposes the flags
// required by the new {appId, manifestId, manifestVersion} payload shape.
// Until d063441 landed on the server, the deploy endpoint accepted inline
// YAML and an optional manifest version. The new shape requires a
// manifest version >= 1 — the validation happens at Run() time, but the
// flag must at least exist on the command tree.
func TestNewCreate_RequiredFlags(t *testing.T) {
	cmd := NewCreate()
	for _, name := range []string{"app-id", "manifest-id", "manifest-version", "wait", "wait-timeout"} {
		require.NotNil(t, cmd.Flags().Lookup(name), "flag %q must be defined", name)
	}
}

func TestNewCreate_WaitTimeoutDefault(t *testing.T) {
	cmd := NewCreate()
	f := cmd.Flags().Lookup("wait-timeout")
	require.NotNil(t, f)
	require.Equal(t, "30m", f.DefValue,
		"a sensible default keeps the previous unbounded-poll behaviour from coming back")
}

// Status constants must stay in sync with the server's deployment status
// enum (terraform-hcp-proxy `internal/storage/models/deployment.go`).
// Renaming any of these on the server without lifting the constants here
// would silently break the wait loop — these assertions are the canary.
func TestStatusConstants(t *testing.T) {
	require.Equal(t, "applied", statusApplied)
	require.Equal(t, "planned_and_finished", statusPlannedAndFinished)
	require.Equal(t, "errored", statusErrored)
}
