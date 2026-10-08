package cmd_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
)

const presentationPayload = `{"data":[{"id":"ledger-1","name":"books","amount":90071992547409930001}],"hasMore":false,"next":"opaque-next"}`

func presentationLedgerServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/prefix/v3/" {
			t.Errorf("unexpected presentation request: %s %s", r.Method, r.URL.RequestURI())
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous presentation request sent credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, presentationPayload); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestPresentationCLINonTerminalJSONContract(t *testing.T) {
	t.Parallel()
	server := presentationLedgerServer(t)
	var want bytes.Buffer
	if err := json.Indent(&want, []byte(presentationPayload), "", "  "); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
	}{
		{"default buffer is JSON", nil},
		{"explicit JSON never colored", []string{"-o", "json", "--color", "always"}},
		{"auto buffer is JSON", []string{"-o", "auto", "--color", "always"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL + "/prefix"}, tc.flags...)
			out, stderr, err := executeRoot(t, append(args, "ledger", "list")...)
			if err != nil || stderr != "" || out != want.String()+"\n" || strings.Contains(out, "\x1b") {
				t.Fatalf("machine JSON changed: stdout=%q stderr=%q err=%v", out, stderr, err)
			}
		})
	}
}

func TestPresentationCLITableColorIsExplicit(t *testing.T) {
	t.Parallel()
	server := presentationLedgerServer(t)
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL + "/prefix", "-otable"}
	plain, stderr, err := executeRoot(t, append(args, "--color", "never", "ledger", "list")...)
	if err != nil || stderr != "" || strings.Contains(plain, "\x1b") || json.Valid([]byte(plain)) {
		t.Fatalf("plain table: stdout=%q stderr=%q err=%v", plain, stderr, err)
	}
	assertPresentationFields(t, plain, "books", "ledger-1", "90071992547409930001", "opaque-next", "false")
	colored, stderr, err := executeRoot(t, append(args, "--color", "always", "ledger", "list")...)
	if err != nil || stderr != "" || !strings.Contains(colored, "\x1b[") {
		t.Fatalf("colored table missing styles: stderr=%q err=%v", stderr, err)
	}
	if stripPresentationANSI(colored) != plain {
		t.Fatal("color changed table fields or layout")
	}
}

func TestPresentationCLIRejectsOptionsBeforeMutation(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"data":{"id":1}}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	for _, flags := range [][]string{{"-o", "yaml"}, {"--output", ""}, {"--color", "invalid"}, {"--color", ""}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			args := append([]string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL}, flags...)
			args = append(args, "ledger", "--ledger", "books", "transactions", "create", "--data", `{"postings":[{"source":"world","destination":"users:1","asset":"USD/2","amount":1}]}`)
			out, stderr, err := executeRoot(t, args...)
			if err == nil || out != "" || stderr != "" || requests.Load() != 0 {
				t.Fatalf("invalid presentation reached service: requests=%d stdout=%q stderr=%q err=%v", requests.Load(), out, stderr, err)
			}
		})
	}
}

func TestPresentationCLIRejectsOptionsBeforeConnectionWrite(t *testing.T) {
	t.Parallel()
	for _, flags := range [][]string{{"-o", "yaml"}, {"--color", "invalid"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			dir := t.TempDir()
			args := append([]string{"--config-dir", dir}, flags...)
			args = append(args, "connections", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000")
			out, stderr, err := executeRoot(t, args...)
			if err == nil || out != "" || stderr != "" {
				t.Fatalf("invalid presentation allowed a local write: stdout=%q stderr=%q err=%v", out, stderr, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "connections.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid presentation changed the connection store: %v", err)
			}
		})
	}
}

func TestPresentationCLIStyledHelpGroupsAndNestedPaths(t *testing.T) {
	t.Parallel()
	out, stderr, err := executeRoot(t, "--color", "always", "--help")
	if err != nil || stderr != "" {
		t.Fatalf("styled help: stderr=%q err=%v", stderr, err)
	}
	for _, group := range []string{"Cloud:", "Modules:", "Connections:"} {
		assertPresentationFields(t, out, "\x1b[1;36m"+group+"\x1b[0m")
	}
	assertPresentationFields(t, stripPresentationANSI(out), "cloud", "ledger", "auth", "connections", "login", "logout")
	for _, path := range [][]string{{"cloud", "stack"}, {"cloud", "stack", "modules"}, {"cloud", "stack", "create"}} {
		args := append([]string{"--color", "never"}, path...)
		out, stderr, err := executeRoot(t, append(args, "--help")...)
		if err != nil || stderr != "" || strings.Contains(out, "\x1b") {
			t.Fatalf("nested help %v: stderr=%q err=%v", path, stderr, err)
		}
		assertPresentationFields(t, out, "fctl "+strings.Join(path, " "), "Usage:")
	}
}

func TestPresentationCLIStackVersionFlagIsSeparateFromRoot(t *testing.T) {
	t.Parallel()
	root := cmd.NewRootCommand()
	root.InitDefaultVersionFlag()
	create, _, err := root.Find([]string{"cloud", "stack", "create"})
	if err != nil {
		t.Fatal(err)
	}
	rootVersion, stackVersion := root.Flags().Lookup("version"), create.Flags().Lookup("version")
	if rootVersion == nil || stackVersion == nil {
		t.Fatal("version flags missing")
	}
	if rootVersion.Value.Type() != "bool" || stackVersion.Value.Type() != "string" || stackVersion.DefValue != "v4.0" {
		t.Fatal("stack catalog version collided with root version flag")
	}
	out, stderr, err := executeRoot(t, "cloud", "stack", "create", "--version", "v4.1", "--help")
	if err != nil || stderr != "" {
		t.Fatalf("stack version string flag: stderr=%q err=%v", stderr, err)
	}
	assertPresentationFields(t, out, "fctl cloud stack create", "--version string")
	out, stderr, err = executeRoot(t, "--version")
	if err != nil || stderr != "" || !strings.HasPrefix(out, "fctl version ") || strings.Contains(out, "Usage:") {
		t.Fatalf("root version flag changed: stdout=%q stderr=%q err=%v", out, stderr, err)
	}
}

func assertPresentationFields(t *testing.T, out string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(out, value) {
			t.Errorf("output missing %q: %s", value, out)
		}
	}
}

func stripPresentationANSI(out string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(out, "")
}
