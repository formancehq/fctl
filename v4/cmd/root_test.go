package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func executeCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	command := NewRootCommand("test-version")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs(args)

	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

func TestRootHelp(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "--help")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, expected := range []string{
		"Formance Control CLI v4",
		"--context",
		"--config-dir",
		"--non-interactive",
		"version",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "fctl v4 test-version\n" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}
