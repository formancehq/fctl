package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	v4config "github.com/formancehq/fctl/v4/internal/config"
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

func TestContextCreateListShowUse(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context local created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CurrentContext != "local" {
		t.Fatalf("expected current context local, got %q", cfg.CurrentContext)
	}
	if cfg.Contexts["local"].Defaults["ledger"] != "default" {
		t.Fatalf("expected default ledger to be persisted")
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "list")
	if err != nil {
		t.Fatalf("list contexts: %v stderr=%s", err, stderr)
	}
	if stdout != "* local\n" {
		t.Fatalf("unexpected list output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "show", "local")
	if err != nil {
		t.Fatalf("show context: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Name: local") || !strings.Contains(stdout, "Kind: stack") {
		t.Fatalf("unexpected show output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "other",
		"--stack-url", "http://other/api",
	)
	if err != nil {
		t.Fatalf("create second context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context other created.\n" {
		t.Fatalf("unexpected second create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "use", "other")
	if err != nil {
		t.Fatalf("use context: %v stderr=%s", err, stderr)
	}
	if stdout != "Current context set to other.\n" {
		t.Fatalf("unexpected use output: %q", stdout)
	}

	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config after use: %v", err)
	}
	if cfg.CurrentContext != "other" {
		t.Fatalf("expected current context other, got %q", cfg.CurrentContext)
	}
}

func TestContextCommandsJSON(t *testing.T) {
	configDir := t.TempDir()

	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
		"--auth-method", "client_credentials",
		"--issuer-url", "http://localhost/api/auth",
		"--client-id", "testing",
		"--secret-ref", "keyring://local/testing",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "context", "list")
	if err != nil {
		t.Fatalf("list contexts json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{`"currentContext": "local"`, `"contexts": [`, `"local"`} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected JSON output to contain %q, got:\n%s", expected, stdout)
		}
	}
}
