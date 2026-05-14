package cmd

import (
	"bytes"
	"fmt"
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
