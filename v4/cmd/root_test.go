package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(body)
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

func TestTargetInspect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Context: local",
		"Target: " + server.URL + "/api (stack)",
		"- ledger 2.3.4 healthy api=[v1 v2] policy=latest-compatible",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected inspect output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestTargetInspectJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		`"context": "local"`,
		`"targetKind": "stack"`,
		`"name": "ledger"`,
		`"apiVersions": [`,
		`"v2"`,
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected JSON inspect output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("includeDeleted"); got != "true" {
				t.Fatalf("expected includeDeleted true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"name":"default","bucket":"bucket","addedAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "list",
		"--page-size", "10",
		"--include-deleted",
	)
	if err != nil {
		t.Fatalf("list ledgers: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"default\tbucket\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger list output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerInfoUsesCanonicalCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "info")
	if err != nil {
		t.Fatalf("read ledger info: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected canonical command to keep stderr empty, got:\n%s", stderr)
	}
	for _, expected := range []string{
		"API version: v1",
		"Server\tledger",
		"Version\t1.9.0",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected info output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerServerInfosDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "ledger", "server-infos")
	if err != nil {
		t.Fatalf("read ledger info through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use ledger info") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "ledger", "create", "default")
	if err == nil {
		t.Fatal("expected ledger create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "ledger create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestLedgerCreateSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"bucket":"bucket-a"`,
				`"features":{"hash":"true"}`,
				`"metadata":{"tier":"gold"}`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "create", "default",
		"--bucket", "bucket-a",
		"--feature", "hash=true",
		"--metadata", "tier=gold",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default created.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/metadata":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"tier":"gold"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "set-metadata", "default", "tier=gold",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set ledger metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/metadata/tier":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "delete-metadata", "default", "tier",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete ledger metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerExportSelectsV2ToFile(t *testing.T) {
	export := "log-entry-1\nlog-entry-2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/export":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(w, export)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	outputPath := filepath.Join(t.TempDir(), "export.jsonl")
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "export",
		"--file", outputPath,
	)
	if err != nil {
		t.Fatalf("export ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default exported to " + outputPath + ".",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if string(data) != export {
		t.Fatalf("unexpected export data: %q", string(data))
	}
}

func TestLedgerExportSelectsV2ToStdout(t *testing.T) {
	export := "log-entry-1\nlog-entry-2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/export":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(w, export)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "export",
		"--file", "-",
	)
	if err != nil {
		t.Fatalf("export ledger: %v stderr=%s", err, stderr)
	}
	if stdout != export {
		t.Fatalf("unexpected stdout export: %q", stdout)
	}
}

func TestLedgerImportSelectsV2FromFile(t *testing.T) {
	input := "{\"id\":1}\n{\"id\":2}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/import":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if body != input {
				t.Fatalf("unexpected import body: %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	inputPath := filepath.Join(t.TempDir(), "import.jsonl")
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write import file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "import", "default",
		"--file", inputPath,
	)
	if err != nil {
		t.Fatalf("import ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default imported.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerImportAcceptsDeprecatedPositionalFile(t *testing.T) {
	input := "{\"id\":1}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/import":
			if body := readRequestBody(t, r); body != input {
				t.Fatalf("unexpected import body: %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	inputPath := filepath.Join(t.TempDir(), "import.jsonl")
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write import file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "import", "default", inputPath,
	)
	if err != nil {
		t.Fatalf("import ledger: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Ledger default imported.") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestLedgerSchemasListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas":
			if got := r.URL.Query().Get("pageSize"); got != "15" {
				t.Fatalf("expected pageSize 15, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{}}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "schemas", "list")
	if err != nil {
		t.Fatalf("list schemas: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "v1\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected schemas output:\n%s", stdout)
	}
}

func TestLedgerSchemasShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas/v1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "schemas", "show", "v1")
	if err != nil {
		t.Fatalf("show schema: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Version\tv1",
		"Created at\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSchemasInsertSelectsV2(t *testing.T) {
	schema := `{"chart":{}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas/v1":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "schema-key" {
				t.Fatalf("expected idempotency key, got %q", got)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"chart":{}`) {
				t.Fatalf("expected schema body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	schemaPath := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "schemas", "insert", "v1",
		"--file", schemaPath,
		"--idempotency-key", "schema-key",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("insert schema: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Schema v1 inserted in ledger default.") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestLedgerAccountsListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts":
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"address":"users:123"`) {
				t.Fatalf("expected query to contain account address, got %q", query)
			}
			if !strings.Contains(query, `"metadata[tier]":"gold"`) {
				t.Fatalf("expected query to contain metadata filter, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"address":"users:123","metadata":{"tier":"gold"}}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "list",
		"--account", "users:123",
		"--metadata", "tier=gold",
	)
	if err != nil {
		t.Fatalf("list accounts: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "users:123") {
		t.Fatalf("unexpected accounts output:\n%s", stdout)
	}
}

func TestLedgerAccountsListDeprecatedAddressAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "list",
		"--address", "users:123",
	)
	if err != nil {
		t.Fatalf("list accounts with deprecated alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Flag --address has been deprecated, use --account") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerAccountsShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"address":"users:123","metadata":{"tier":"gold"},"volumes":{"USD/2":{"input":100,"output":40,"balance":60}}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "accounts", "show", "users:123")
	if err != nil {
		t.Fatalf("show account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Address\tusers:123",
		"USD/2\tinput=100\toutput=40\tbalance=60",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerAccountsSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123/metadata":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"foo":"bar"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "set-metadata", "users:123", "foo=bar",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set account metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerAccountsDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123/metadata/foo":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "delete-metadata", "users:123", "foo",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete account metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerStatsSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/stats":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"accounts":2,"transactions":42}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "stats")
	if err != nil {
		t.Fatalf("read stats: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Transactions\t42",
		"Accounts\t2",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected stats output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if got := r.URL.Query().Get("pageSize"); got != "15" {
				t.Fatalf("expected pageSize 15, got %q", got)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			if !strings.Contains(query, `"metadata[tier]":"gold"`) {
				t.Fatalf("expected query to contain metadata tier, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":1,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--source", "world",
		"--destination", "users:123",
		"--metadata", "tier=gold",
	)
	if err != nil {
		t.Fatalf("list transactions: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"1\tref\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsListTimeFiltersSelectV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/default/transactions":
			if got := r.URL.Query().Get("startTime"); got != "2026-01-01T00:00:00Z" {
				t.Fatalf("expected startTime, got %q", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "2026-01-02T00:00:00Z" {
				t.Fatalf("expected endTime, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"txid":1,"metadata":{"foo":"bar"},"postings":[],"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--start", "2026-01-01T00:00:00Z",
		"--end", "2026-01-02T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("list transactions: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") {
		t.Fatalf("expected v1 output, got:\n%s", stdout)
	}
}

func TestLedgerTransactionsCountSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if r.Method != http.MethodHead {
				t.Fatalf("expected HEAD, got %s", r.Method)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			w.Header().Set("Count", "42")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "count",
		"--source", "world",
		"--destination", "users:123",
	)
	if err != nil {
		t.Fatalf("count transactions: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Count\t42",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsSendSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"amount":100`,
				`"asset":"USD/2"`,
				`"source":"world"`,
				`"destination":"users:123"`,
				`"metadata":{"foo":"bar"}`,
				`"reference":"ref"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "send",
		"--source", "world",
		"--destination", "users:123",
		"--amount", "100",
		"--asset", "USD/2",
		"--metadata", "foo=bar",
		"--reference", "ref",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("send transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID\t42",
		"Reference\tref",
		"Timestamp\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSendDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"source":"world"`,
				`"destination":"users:123"`,
				`"amount":100`,
				`"asset":"USD/2"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "send",
		"users:123", "100", "USD/2",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("send through deprecated alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use ledger transactions send") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Positional ledger send arguments have been deprecated") {
		t.Fatalf("expected positional deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerTransactionsShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "transactions", "show", "42")
	if err != nil {
		t.Fatalf("show transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID\t42",
		"Reference\tref",
		"Timestamp\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected transaction output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsRevertRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "ledger", "transactions", "revert", "42")
	if err == nil {
		t.Fatal("expected revert to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "ledger transactions revert requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestLedgerTransactionsRevertSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/revert":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.URL.Query().Get("atEffectiveDate"); got != "true" {
				t.Fatalf("expected atEffectiveDate true, got %q", got)
			}
			if got := r.URL.Query().Get("force"); got != "true" {
				t.Fatalf("expected force true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":43,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"revert-ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "revert", "42",
		"--at-effective-date",
		"--force",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("revert transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID\t43",
		"Reference\trevert-ref",
		"Timestamp\t2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected transaction output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/metadata":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"foo":"bar"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "set-metadata", "42", "foo=bar",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set transaction metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/metadata/foo":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "delete-metadata", "42", "foo",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete transaction metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerVolumesListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/volumes":
			if got := r.URL.Query().Get("startTime"); got != "2026-01-01T00:00:00Z" {
				t.Fatalf("expected startTime, got %q", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "2026-01-02T00:00:00Z" {
				t.Fatalf("expected endTime, got %q", got)
			}
			if got := r.URL.Query().Get("insertionDate"); got != "true" {
				t.Fatalf("expected insertionDate true, got %q", got)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"account":"users:123"`) {
				t.Fatalf("expected query to contain account, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"account":"users:123","asset":"USD/2","input":100,"output":40,"balance":60}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "volumes", "list",
		"--account", "users:123",
		"--start-time", "2026-01-01T00:00:00Z",
		"--end-time", "2026-01-02T00:00:00Z",
		"--use-insertion-date",
	)
	if err != nil {
		t.Fatalf("list volumes: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"users:123\tUSD/2\t100\t40\t60",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected volumes output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerVolumesListDeprecatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/volumes":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "volumes", "list",
		"--address", "users:123",
		"--oot", "2026-01-01T00:00:00Z",
		"--pit", "2026-01-02T00:00:00Z",
		"--insertion-date",
	)
	if err != nil {
		t.Fatalf("list volumes with deprecated aliases: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Flag --address has been deprecated, use --account",
		"Flag --oot has been deprecated, use --start-time",
		"Flag --pit has been deprecated, use --end-time",
		"Flag --insertion-date has been deprecated, use --use-insertion-date",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected stderr to contain %q, got:\n%s", expected, stderr)
		}
	}
}

func TestLedgerTransactionsListDeprecatedSourceDestinationAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--src", "world",
		"--dst", "users:123",
	)
	if err != nil {
		t.Fatalf("list transactions with deprecated aliases: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Flag --src has been deprecated, use --source",
		"Flag --dst has been deprecated, use --destination",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected stderr to contain %q, got:\n%s", expected, stderr)
		}
	}
}

func TestLedgerTransactionsListJSON(t *testing.T) {
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
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "ledger", "transactions", "list")
	if err != nil {
		t.Fatalf("list transactions json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		`"apiVersion": "v2"`,
		`"transactions": []`,
		`"pageSize": 15`,
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger JSON output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsAccountsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{"env":"dev"},"raw":{},"name":"Main"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payment accounts: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"acc_1\tref\t2026-01-01T00:00:00Z\tMain\tUSD/2\tconn_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payments accounts output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsAccountsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{"env":"dev"},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "show", "acc_1",
	)
	if err != nil {
		t.Fatalf("show payment account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tacc_1",
		"Reference\tref",
		"Name\tMain",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsAccountsGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "get", "acc_1",
	)
	if err != nil {
		t.Fatalf("show payment account through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments accounts show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsAccountsBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1/balances":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("asset"); got != "USD/2" {
				t.Fatalf("expected asset USD/2, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"accountID":"acc_1","asset":"USD/2","balance":100,"createdAt":"2026-01-01T00:00:00Z","lastUpdatedAt":"2026-01-02T00:00:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "balances", "acc_1",
		"--asset", "USD/2",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list account balances: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"acc_1\tUSD/2\t100\t2026-01-01T00:00:00Z\t2026-01-02T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected account balances output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsBankAccountsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"ba_1","name":"Main","createdAt":"2026-01-01T00:00:00Z","country":"FR","metadata":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list bank accounts: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "ba_1\tMain\t2026-01-01T00:00:00Z\tFR") {
		t.Fatalf("unexpected bank accounts output:\n%s", stdout)
	}
}

func TestPaymentsBankAccountsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts/ba_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"ba_1","name":"Main","createdAt":"2026-01-01T00:00:00Z","country":"FR","iban":"FR7630006000011234567890189","swiftBicCode":"BNPAFRPP","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "show", "ba_1",
	)
	if err != nil {
		t.Fatalf("show bank account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tba_1",
		"Country\tFR",
		"IBAN\tFR7630006000011234567890189",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected bank account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsBankAccountsDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank_accounts", "list",
	)
	if err != nil {
		t.Fatalf("list bank accounts through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments bank-accounts") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsPaymentsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payments: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "pay_1\tPAYOUT\t100\tUSD/2\tSUCCEEDED\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected payments output:\n%s", stdout)
	}
}

func TestPaymentsPaymentsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments/pay_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "show", "pay_1",
	)
	if err != nil {
		t.Fatalf("show payment: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tpay_1",
		"Reference\tref",
		"Amount\t100",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"pool_1","name":"Main","poolAccounts":["acc_1"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payment pools: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "pool_1\tMain\tacc_1") {
		t.Fatalf("unexpected payment pools output:\n%s", stdout)
	}
}

func TestPaymentsPoolsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pool_1","name":"Main","poolAccounts":["acc_1","acc_2"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "show", "pool_1",
	)
	if err != nil {
		t.Fatalf("show payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tpool_1",
		"Name\tMain",
		"Accounts\tacc_1,acc_2",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pool_1","name":"Main","poolAccounts":["acc_1"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "get", "pool_1",
	)
	if err != nil {
		t.Fatalf("show payment pool through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments pools show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsPoolsDeleteRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "pools", "delete", "pool_1")
	if err == nil {
		t.Fatal("expected payments pools delete to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments pools delete requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsDeleteSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "delete", "pool_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Pool pool_1 deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool delete output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsAddAccountSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/accounts/acc_1":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "add-account", "pool_1", "acc_1",
	)
	if err != nil {
		t.Fatalf("add account to payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Account acc_1 added to pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool add-account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsRemoveAccountRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "pools", "remove-account", "pool_1", "acc_1")
	if err == nil {
		t.Fatal("expected payments pools remove-account to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments pools remove-account requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsRemoveAccountSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/accounts/acc_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "remove-account", "pool_1", "acc_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("remove account from payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Account acc_1 removed from pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool remove-account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsUpdateQuerySelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/query":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"query":{"accountID":"acc_1"}`) {
				t.Fatalf("expected query body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	queryPath := filepath.Join(t.TempDir(), "query.json")
	if err := os.WriteFile(queryPath, []byte(`{"query":{"accountID":"acc_1"}}`), 0o600); err != nil {
		t.Fatalf("write query fixture: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "update-query", "pool_1",
		"--file", queryPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update pool query: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Query updated for pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool update-query output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/balances":
			if got := r.URL.Query().Get("at"); got == "" {
				t.Fatal("expected at query parameter")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"asset":"USD/2","amount":100,"relatedAccounts":["acc_1"]}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "balances", "pool_1",
		"--at", "2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("list pool balances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "USD/2\t100\tacc_1") {
		t.Fatalf("unexpected pool balances output:\n%s", stdout)
	}
}

func TestPaymentsPoolsLatestBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/balances/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"asset":"USD/2","amount":100,"relatedAccounts":["acc_1"]}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "latest-balances", "pool_1",
	)
	if err != nil {
		t.Fatalf("list latest pool balances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "USD/2\t100\tacc_1") {
		t.Fatalf("unexpected latest pool balances output:\n%s", stdout)
	}
}

func TestPaymentsTasksShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/tasks/task_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"task_1","connectorID":"conn_1","createdObjectID":"pay_1","status":"SUCCEEDED","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "tasks", "show", "task_1",
	)
	if err != nil {
		t.Fatalf("show payment task: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\ttask_1",
		"Connector ID\tconn_1",
		"Status\tSUCCEEDED",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment task output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTasksGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/tasks/task_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"task_1","status":"SUCCEEDED","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "tasks", "get", "task_1",
	)
	if err != nil {
		t.Fatalf("show payment task through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments tasks show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestConfigMigrateV3DryRun(t *testing.T) {
	v3Dir := writeV3CommandFixture(t, true)

	stdout, stderr, err := executeCommand(t, "config", "migrate-v3", "--from", v3Dir, "--dry-run")
	if err != nil {
		t.Fatalf("migrate dry-run: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Current context: default",
		"- default (cloud-stack)",
		"Credential moves: 1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected migration output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestConfigMigrateV3Write(t *testing.T) {
	v3Dir := writeV3CommandFixture(t, false)
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "config", "migrate-v3", "--from", v3Dir)
	if err != nil {
		t.Fatalf("migrate write: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Migrated 1 context(s)") {
		t.Fatalf("unexpected migration output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
	if cfg.CurrentContext != "default" || cfg.Contexts["default"].Kind != v4config.ContextKindCloudStack {
		t.Fatalf("unexpected migrated config: %#v", cfg)
	}
}

func writeV3CommandFixture(t *testing.T, withTokens bool) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "profiles", "default"), 0o700); err != nil {
		t.Fatalf("create v3 fixture dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(`{"currentProfile":"default"}`), 0o600); err != nil {
		t.Fatalf("write v3 config: %v", err)
	}
	rootTokens := "null"
	if withTokens {
		rootTokens = `{"access":{"token":"access-token"},"id":{"token":"id-token"}}`
	}
	profile := `{
	  "membershipURI": "https://app.formance.cloud/api",
	  "rootTokens": ` + rootTokens + `,
	  "defaultOrganization": "org_123",
	  "defaultStack": "stack_123"
	}`
	if err := os.WriteFile(filepath.Join(dir, "profiles", "default", "profile.json"), []byte(profile), 0o600); err != nil {
		t.Fatalf("write v3 profile: %v", err)
	}
	return dir
}
