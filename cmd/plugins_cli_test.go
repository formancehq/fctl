package cmd_test

import (
	"io"
	"net/http"
	"net/http/httptest"
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
	name    string
	command []string
	after   string
	reject  bool
}

func TestLedgerPluginCLIPaginationContract(t *testing.T) {
	t.Parallel()
	for _, test := range []ledgerPaginationCase{
		{"all ledgers", []string{"list"}, "", false},
		{"account address", []string{"accounts", "list", "--after", "users:next/+="}, "users:next/+=", false},
		{"transaction ID", []string{"transactions", "list", "--after", "18446744073709551615"}, "18446744073709551615", false},
		{"log ID", []string{"logs", "list", "--after", "9007199254740993"}, "9007199254740993", false},
		{"ledger page size", []string{"list", "--page-size", "1"}, "", true},
		{"ledger reverse", []string{"list", "--reverse"}, "", true},
		{"ledger cursor", []string{"list", "--cursor", "next"}, "", true},
		{"log reverse", []string{"logs", "list", "--reverse"}, "", true},
		{"account cursor", []string{"accounts", "list", "--cursor", "next"}, "", true},
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
		query := r.URL.Query()
		if query.Get("after") != test.after || query.Has("cursor") || query.Has("reverse") {
			t.Errorf("pagination query: %v", query)
		}
		if test.name == "all ledgers" && len(query) != 0 {
			t.Errorf("list all query: %v", query)
		}
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
			t.Fatalf("unsupported flag reached HTTP: out=%s calls=%d err=%v", out, calls.Load(), err)
		}
		return
	}
	if err != nil || calls.Load() != 1 {
		t.Fatalf("pagination failed: calls=%d err=%v", calls.Load(), err)
	}
	assertCLICloudJSON(t, out, response)
}
