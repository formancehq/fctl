package cmd_test

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLedgerPluginCLIWriteAndPartialFailure(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, command, path, body, response string
		status                              int
		wantError                           bool
	}{
		{"create", "transactions", "/v3/books/transactions", `{"postings":[{"source":"world","destination":"users:001","asset":"USD/2","amount":9007199254740993}]}`, `{"data":{"id":1,"amount":9007199254740993}}`, 200, false},
		{"bulk business failure", "bulk", "/v3/books/bulk", `[]`, `{"data":[{"responseType":"ERROR","errorCode":"VALIDATION"}]}`, 200, true},
		{"bulk HTTP failure", "bulk", "/v3/books/bulk", `[]`, `{"errorCode":"VALIDATION","errorMessage":"batch failed","data":[{"responseType":"ERROR"}]}`, 400, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assertLedgerPluginWrite(t, r, test.path, test.body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				if _, err := io.WriteString(w, test.response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL, "ledger", "--ledger", "books", test.command}
			if test.command == "transactions" {
				args = append(args, "create")
			}
			args = append(args, "--data", test.body, "--idempotency-key", "logical-write")
			out, stderr, err := executeRoot(t, args...)
			if (err != nil) != test.wantError {
				t.Fatalf("command error = %v", err)
			}
			assertCLICloudJSON(t, out, test.response)
			if stderr != "" || calls.Load() != 1 {
				t.Fatalf("stderr=%s calls=%d", stderr, calls.Load())
			}
		})
	}
}

func assertLedgerPluginWrite(t *testing.T, r *http.Request, path, body string) {
	t.Helper()
	if r.Method != http.MethodPost || r.URL.Path != path || r.Header.Get("Idempotency-Key") != "logical-write" {
		t.Errorf("unexpected mutation: %s %s, key=%s", r.Method, r.URL.Path, r.Header.Get("Idempotency-Key"))
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if strings.TrimSpace(string(data)) != body {
		t.Errorf("body changed: %s", data)
	}
}

type ledgerPaginationCase struct {
	name     string
	command  []string
	pageSize string
	after    string
	reverse  string
	cursor   string
	reject   bool
}

func TestLedgerPluginCLIPaginationContract(t *testing.T) {
	t.Parallel()
	for _, test := range []ledgerPaginationCase{
		{name: "ledger page", command: []string{"list"}, pageSize: "100"},
		{name: "account address", command: []string{"accounts", "list", "--after", "users:next/+="}, pageSize: "100", after: "users:next/+="},
		{name: "transaction ID", command: []string{"transactions", "list", "--after", "18446744073709551615"}, pageSize: "100", after: "18446744073709551615"},
		{name: "log ID", command: []string{"logs", "list", "--after", "9007199254740993"}, pageSize: "100", after: "9007199254740993"},
		{name: "ledger page size", command: []string{"list", "--page-size", "1"}, pageSize: "1"},
		{name: "ledger reverse", command: []string{"list", "--reverse"}, pageSize: "100", reverse: "true"},
		{name: "log reverse", command: []string{"logs", "list", "--reverse"}, pageSize: "100", reverse: "true"},
		{name: "ledger cursor", command: []string{"list", "--cursor", "eyJrZXkiOiJuZXh0In0"}, pageSize: "100", cursor: "eyJrZXkiOiJuZXh0In0"},
		{name: "account cursor", command: []string{"accounts", "list", "--cursor", "eyJrZXkiOiJuZXh0In0"}, pageSize: "100", cursor: "eyJrZXkiOiJuZXh0In0"},
		{name: "invalid ledger cursor", command: []string{"list", "--cursor", "next"}, reject: true},
		{name: "invalid account cursor", command: []string{"accounts", "list", "--cursor", "next"}, reject: true},
		{name: "after and cursor conflict", command: []string{"accounts", "list", "--after", "bank", "--cursor", "eyJrZXkiOiJuZXh0In0"}, reject: true},
	} {
		t.Run(test.name, func(t *testing.T) { checkLedgerPaginationCLI(t, test) })
	}
}

func checkLedgerPaginationCLI(t *testing.T, test ledgerPaginationCase) {
	t.Helper()
	var calls atomic.Int32
	const response = `{"data":[{"id":9007199254740993,"amount":1234567890123456789012345678901234567890}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assertLedgerPaginationQuery(t, test, r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL, "--output", "json", "ledger", "--ledger", "books"}
	out, _, err := executeRoot(t, append(args, test.command...)...)
	if test.reject {
		if err == nil || calls.Load() != 0 || out != "" {
			t.Fatalf("invalid pagination reached HTTP: out=%s calls=%d err=%v", out, calls.Load(), err)
		}
		return
	}
	if err != nil || calls.Load() != 1 {
		t.Fatalf("pagination failed: calls=%d err=%v", calls.Load(), err)
	}
	assertCLICloudJSON(t, out, response)
}

func assertLedgerPaginationQuery(t *testing.T, test ledgerPaginationCase, query url.Values) {
	t.Helper()
	if query.Get("pageSize") != test.pageSize || query.Get("reverse") != test.reverse || query.Has("after") {
		t.Errorf("pagination query: %v", query)
	}
	if test.after != "" {
		assertLedgerAfterCursor(t, query.Get("cursor"), test.after)
	} else if query.Get("cursor") != test.cursor {
		t.Errorf("cursor changed: %v", query)
	}
	for key := range query {
		if key != "pageSize" && key != "cursor" && key != "reverse" {
			t.Errorf("unexpected pagination parameter %q", key)
		}
	}
}

func assertLedgerAfterCursor(t *testing.T, cursor, want string) {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		t.Fatal(err)
	}
	var token struct {
		Key  string `json:"key"`
		Back bool   `json:"back"`
	}
	if err := json.Unmarshal(raw, &token); err != nil {
		t.Fatal(err)
	}
	if token.Key != want || token.Back {
		t.Errorf("after cursor: %s", raw)
	}
}
