package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIIntegrationCoreWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()

	result := runCLI(t,
		"--config-dir", configDir,
		"--non-interactive",
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	result.requireSuccess(t)
	if !strings.Contains(result.stdout, "Context local created.") {
		t.Fatalf("unexpected create output: %q", result.stdout)
	}

	result = runCLI(t, "--config-dir", configDir, "-o", "yaml", "context", "list")
	result.requireSuccess(t)
	for _, expected := range []string{"currentContext: local", "- local"} {
		if !strings.Contains(result.stdout, expected) {
			t.Fatalf("expected YAML output to contain %q, got:\n%s", expected, result.stdout)
		}
	}

	result = runCLI(t, "--config-dir", configDir, "-o", "json", "target", "inspect")
	result.requireSuccess(t)
	for _, expected := range []string{`"targetKind": "stack"`, `"name": "ledger"`, `"v2"`} {
		if !strings.Contains(result.stdout, expected) {
			t.Fatalf("expected inspect JSON to contain %q, got:\n%s", expected, result.stdout)
		}
	}

	result = runCLI(t, "--config-dir", configDir, "-o", "json", "ledger", "transactions", "list")
	result.requireSuccess(t)
	for _, expected := range []string{`"apiVersion": "v2"`, `"transactions": []`} {
		if !strings.Contains(result.stdout, expected) {
			t.Fatalf("expected ledger JSON to contain %q, got:\n%s", expected, result.stdout)
		}
	}

	result = runCLI(t, "--config-dir", configDir, "ledger", "transactions", "list", "--api-version", "v3")
	if result.exitCode == 0 {
		t.Fatalf("expected pinned unsupported API version to fail")
	}
	if !strings.Contains(result.stderr, "does not support pinned api version v3") {
		t.Fatalf("expected unsupported api error, got stderr:\n%s", result.stderr)
	}
}

func TestCLIIntegrationMissingConfigError(t *testing.T) {
	result := runCLI(t, "--config-dir", t.TempDir(), "target", "inspect")
	if result.exitCode == 0 {
		t.Fatalf("expected missing config to fail")
	}
	if !strings.Contains(result.stderr, "read config") && !strings.Contains(result.stderr, "no such file") {
		t.Fatalf("unexpected stderr:\n%s", result.stderr)
	}
}

func TestCLIIntegrationInvalidConfigError(t *testing.T) {
	result := runCLI(t,
		"--config-dir", t.TempDir(),
		"context", "create", "stack", "local",
	)
	if result.exitCode == 0 {
		t.Fatalf("expected invalid context creation to fail")
	}
	if !strings.Contains(result.stderr, "stackURL is required") {
		t.Fatalf("unexpected stderr:\n%s", result.stderr)
	}
}

func TestCLIIntegrationMissingAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("request should not reach server without credentials")
	}))
	defer server.Close()

	configDir := t.TempDir()
	result := runCLI(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--auth-method", "client_credentials",
		"--issuer-url", server.URL,
		"--client-id", "client",
		"--secret-ref", "missing-secret",
	)
	result.requireSuccess(t)

	result = runCLI(t, "--config-dir", configDir, "target", "inspect")
	if result.exitCode == 0 {
		t.Fatalf("expected missing auth to fail")
	}
	if !strings.Contains(result.stderr, "credential not found") {
		t.Fatalf("unexpected stderr:\n%s", result.stderr)
	}
}

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r cliResult) requireSuccess(t *testing.T) {
	t.Helper()
	if r.exitCode != 0 {
		t.Fatalf("expected success, got exit %d\nstdout:\n%s\nstderr:\n%s", r.exitCode, r.stdout, r.stderr)
	}
	if r.stderr != "" {
		t.Fatalf("expected empty stderr, got:\n%s", r.stderr)
	}
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()

	commandArgs := append([]string{"run", "."}, args...)
	cmd := exec.Command("go", commandArgs...)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}
