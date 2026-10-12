package integration_test

import (
	"bytes"
	"net/http"
	"slices"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
)

func executeRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := cmd.NewRootCommand()
	// Service metadata comes from the selected target's external plugin cache.
	if slices.Contains(args, "auth") || slices.Contains(args, "ledger") {
		root = cmd.NewRootCommandWithArgs(t.Context(), args)
	}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(append([]string{"--no-browser"}, args...))
	err := root.ExecuteContext(t.Context())
	return stdout.String(), stderr.String(), err
}

func writeCLIServiceInfo(t *testing.T, w http.ResponseWriter, version string) {
	t.Helper()
	// The SDK guard reads the service's full API version from the info envelope.
	writeCLICloudJSON(t, w, map[string]any{"data": map[string]string{"version": version}})
}
