package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectionLifecycle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, stderr, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		if stderr != "" {
			t.Fatalf("unexpected stderr: %s", stderr)
		}
		if !json.Valid([]byte(out)) {
			t.Fatalf("invalid JSON: %s", out)
		}
		return out
	}
	run("connections", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000")
	run("connections", "add", "other", "--auth-mode", "none", "--ledger-url", "http://localhost:9001")
	run("connections", "use", "other")
	if out := run("connections", "show"); !strings.Contains(out, "9001") {
		t.Fatal(out)
	}
	if out := run("--connection", "local", "connections", "show"); !strings.Contains(out, "9000") {
		t.Fatal(out)
	}
	if out := run("connections", "list"); !strings.Contains(out, `"loggedIn": false`) {
		t.Fatal(out)
	}
	run("connections", "add", "other", "--replace", "--auth-mode", "none", "--ledger-url", "http://localhost:9002")
	run("connections", "delete", "other", "--confirm")
	if out := run("connections", "list"); strings.Contains(out, `"name": "other"`) {
		t.Fatal(out)
	}
	info, err := os.Stat(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode: %v", info.Mode())
	}
}

func TestConnectionErrorsAndLoginBoundaries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, _, err := executeRoot(t, "--config-dir", dir, "connections", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"connections", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"},
		{"connections", "add", "../escape", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"},
		{"connections", "delete", "local"},
		{"connections", "delete", "missing", "--confirm"},
		{"connections", "use", "missing"},
		{"--connection", "missing", "connections", "show"},
		{"login"},
	} {
		if _, _, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...); err == nil {
			t.Fatalf("accepted invalid command %v", args)
		}
	}
	if _, _, err := executeRoot(t, "--config-dir", dir, "logout"); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedServicesUseConnection(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous connection sent credentials")
		}
		if r.URL.Path != "/prefix/api/ledger/v3/" && r.URL.Path != "/prefix/api/auth/clients" {
			t.Errorf("unexpected route: %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(`{"data":[]}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	if _, _, err := executeRoot(t, "--config-dir", dir, "connections", "add", "local", "--stack-url", server.URL+"/prefix", "--auth-mode", "none"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"ledger", "list"}, {"auth", "clients", "list"}} {
		out, _, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, `"data": []`) {
			t.Fatal(out)
		}
	}
}
