package cmd_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPluginDependencyClosure guards the extraction point, including transitive
// imports. A service plugin must remain usable without the host CLI and config.
func TestPluginDependencyClosure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-deps", "../plugins/...", "../pkg/pluginsdk/...")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("read plugin dependency closure: %v: %s", err, output)
	}
	for dependency := range strings.FieldsSeq(string(output)) {
		if strings.HasPrefix(dependency, "github.com/formancehq/fctl/v4/internal/") ||
			strings.HasPrefix(dependency, "github.com/formancehq/fctl/v4/cmd") ||
			dependency == "github.com/spf13/cobra" || dependency == "github.com/spf13/pflag" {
			t.Errorf("plugin dependency on host implementation: %s", dependency)
		}
	}
}
