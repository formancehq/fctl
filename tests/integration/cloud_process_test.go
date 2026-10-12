package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
)

// The re-executed race test binary captures both bootstrap and execution output
// without replacing process-wide writers shared by parallel fixture tests.
func executeCLICloudProcess(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	process := exec.CommandContext(t.Context(), binary, append([]string{"-test.run=^TestCLICloudProcess$", "--"}, args...)...) //nolint:gosec // Re-executes the current test binary with test-owned CLI arguments.
	process.Env = append(os.Environ(), "FCTL_INTEGRATION_CLOUD_PROCESS=1")
	var stdout, stderr bytes.Buffer
	process.Stdout, process.Stderr = &stdout, &stderr
	err = process.Run()
	return stdout.String(), stderr.String(), err
}

func TestCLICloudProcess(t *testing.T) {
	if os.Getenv("FCTL_INTEGRATION_CLOUD_PROCESS") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator == -1 {
		t.Fatal("CLI helper process requires an argument separator")
	}
	args := os.Args[separator+1:]
	root := cmd.NewRootCommandWithArgs(t.Context(), args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
	// Do not append the Go test runner's PASS line to the CLI's JSON stdout.
	os.Exit(0)
}
