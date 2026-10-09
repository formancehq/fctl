package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd"
)

// Product integration is opt-in: the host must never import or build the
// Connectivity module. Supply a product-built executable at exact version 1.2.3.
func connectivityRoot(t *testing.T, args []string) *cobra.Command {
	t.Helper()
	binary := os.Getenv("FCTL_CONNECTIVITY_TEST_BINARY")
	if binary == "" {
		t.Skip("set FCTL_CONNECTIVITY_TEST_BINARY to product executable built for 1.2.3")
	}
	prefix := []string{}
	for i := 0; i < len(args) && args[i] != "connectivity"; i++ {
		prefix = append(prefix, args[i])
	}
	install := append(append([]string{}, prefix...), "plugins", "install", "--service", "connectivity", "--binary", binary, "--service-version", "1.2.3")
	out, trace, err := executeExternalCLI(t, install)
	if err != nil || !strings.Contains(out, "connectivity") {
		t.Fatalf("install: %s %s %v", out, trace, err)
	}
	// Cached metadata attaches forms without discovery; the actual execution
	// still checks the live service version and uses the host HTTP broker.
	return cmd.NewRootCommandWithArgs(t.Context(), append(prefix, "--help"))
}
