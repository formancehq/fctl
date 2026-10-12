package integration_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/cmd"
)

const largeLegacyLogs = 20_000

func TestLegacyInlineImportWithStreamManifest(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	t.Setenv("FCTL_PROFILE", "")
	api := &legacyStreamAPI{t: t, digest: sha256.New()}
	server := httptest.NewServer(api)
	defer server.Close()
	out, trace, err := executeExternalCLI(t, []string{"--config-dir", t.TempDir(), "--auth-mode", "none",
		"--ledger-url", server.URL, "--no-input", "-o", "json", "legacy", "ledger", "import", "books",
		"--data", `[{"id":1,"payload":{"amount":90071992547409931234567890}}]`})
	if err != nil || trace != "" || !strings.Contains(out, `"imported": 1`) {
		t.Fatalf("inline legacy import changed: stdout=%s stderr=%s err=%v", out, trace, err)
	}
}

func largeLegacySource(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "logs.ndjson")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // Test-owned temporary source.
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	writer := io.MultiWriter(file, digest)
	payload := strings.Repeat("x", 4096)
	for i := range largeLegacyLogs {
		if _, err := fmt.Fprintf(writer, "{\"id\":%d,\"payload\":{\"amount\":90071992547409931234567890,\"unknown\":\"%s\"}}\n", i, payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() <= 64<<20 {
		t.Fatalf("fixture is not a large file: %v %v", info, err)
	}
	return path, digest.Sum(nil)
}

type legacyStreamAPI struct {
	t      *testing.T
	source string
	mu     sync.Mutex
	digest hash.Hash
	logs   int
}

func (s *legacyStreamAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/_info":
		writeCLIServiceInfo(s.t, w, "2.4.15")
	case "/v2/books/logs/import":
		// Bound each batch, while the total source greatly exceeds unary limits.
		body, err := io.ReadAll(io.LimitReader(r.Body, (4<<20)+1))
		if err != nil || len(body) > 4<<20 || r.Method != http.MethodPost {
			s.t.Errorf("invalid stream batch: %d bytes %s %v", len(body), r.Method, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, err := s.digest.Write(body); err != nil {
			s.t.Error(err)
		}
		s.logs += bytes.Count(body, []byte{'\n'})
		w.WriteHeader(http.StatusNoContent)
	case "/v2/books/logs/export":
		file, err := os.Open(s.source)
		if err != nil {
			s.t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				s.t.Error(err)
			}
		}()
		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := io.Copy(w, file); err != nil {
			s.t.Error(err)
		}
	default:
		s.t.Errorf("unexpected legacy request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestLargeLegacyCLIImportExport(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	t.Setenv("FCTL_PROFILE", "")
	source, expected := largeLegacySource(t)
	api := &legacyStreamAPI{t: t, source: source, digest: sha256.New()}
	server := httptest.NewServer(api)
	defer server.Close()
	base := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--ledger-url", server.URL, "--no-input", "-o", "json"}
	args := append(append([]string{}, base...), "legacy", "ledger", "import", "books", source)
	out, trace, err := executeExternalCLI(t, args)
	var result struct {
		Imported int `json:"imported"`
	}
	if err != nil || json.Unmarshal([]byte(out), &result) != nil || result.Imported != largeLegacyLogs || trace != "" {
		t.Fatalf("large import result=%s trace=%s err=%v", out, trace, err)
	}
	api.mu.Lock()
	actual, logs := api.digest.Sum(nil), api.logs
	api.mu.Unlock()
	if logs != largeLegacyLogs || !bytes.Equal(actual, expected) {
		t.Fatalf("large import changed records: logs=%d digest=%x expected=%x", logs, actual, expected)
	}
	destination := filepath.Join(t.TempDir(), "export.ndjson")
	args = append(append([]string{}, base...), "legacy", "ledger", "export", "--ledger", "books", "--file", destination)
	out, trace, err = executeExternalCLI(t, args)
	if err != nil || trace != "" {
		t.Fatalf("large export result=%s trace=%s err=%v", out, trace, err)
	}
	verifyStreamFile(t, destination, expected)
	// A hash writer proves stdout receives only NDJSON, never the final JSON
	// summary, terminal styling or progress. It does not retain the export.
	args = append(append([]string{}, base...), "legacy", "ledger", "export", "--ledger", "books")
	root := cmd.NewRootCommandWithArgs(t.Context(), args)
	digest := sha256.New()
	var stderr bytes.Buffer
	root.SetOut(digest)
	root.SetErr(&stderr)
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err != nil || stderr.Len() != 0 || !bytes.Equal(digest.Sum(nil), expected) {
		t.Fatalf("stdout export changed: digest=%x stderr=%s err=%v", digest.Sum(nil), stderr.String(), err)
	}
	t.Logf("large CLI source: %d logs; file export and stdout SHA-256 verified", largeLegacyLogs)
}

func verifyStreamFile(t *testing.T, path string, expected []byte) {
	t.Helper()
	file, err := os.Open(path) //nolint:gosec // Test-owned temporary export.
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, err = io.Copy(digest, file)
	err = errors.Join(err, file.Close())
	if err != nil || !bytes.Equal(digest.Sum(nil), expected) {
		t.Fatalf("export changed: digest=%x expected=%x err=%v", digest.Sum(nil), expected, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("export permissions: %v %v", info, err)
	}
}

func TestLargeLegacyExecutableImportExport(t *testing.T) {
	source, expected := largeLegacySource(t)
	api := &legacyStreamAPI{t: t, source: source, digest: sha256.New()}
	server := httptest.NewServer(api)
	defer server.Close()
	binary := filepath.Join(t.TempDir(), "fctl-plugin-legacy")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/fctl-plugin-legacy") //nolint:gosec // Fixed build command for the repository-owned plugin.
	build.Dir = filepath.Join("..", "..", "misc", "fctl-plugin")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build legacy executable: %s %v", out, err)
	}
	client, err := transport.Open(t.Context(), binary, server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	}()
	importLegacyExecutableFile(t, client, source, api, expected, server.URL)
	digest := sha256.New()
	updates := 0
	result, err := client.ExecuteStream(t.Context(), pluginsdk.ExecuteRequest{
		CommandPath: []string{"legacy", "ledger", "export"}, Flags: map[string]string{"ledger": "books"}, Endpoint: server.URL,
	}, pluginsdk.ExecutionStreams{Output: digest, Report: func(event pluginsdk.ProgressEvent) error {
		updates++
		return event.Validate()
	}})
	if err != nil || !bytes.Equal(digest.Sum(nil), expected) || updates < 2 || !json.Valid(result.Data) {
		t.Fatalf("native stream digest=%x updates=%d result=%s err=%v", digest.Sum(nil), updates, result.Data, err)
	}
}

func importLegacyExecutableFile(t *testing.T, client *transport.Client, source string, api *legacyStreamAPI, expected []byte, endpoint string) {
	t.Helper()
	file, err := os.Open(source) //nolint:gosec // Test-owned generated source.
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	updates := 0
	result, err := client.ExecuteStream(t.Context(), pluginsdk.ExecuteRequest{
		CommandPath: []string{"legacy", "ledger", "import"}, Args: []string{"books", source}, Endpoint: endpoint,
	}, pluginsdk.ExecutionStreams{Input: file, InputSize: new(info.Size()), Report: func(event pluginsdk.ProgressEvent) error {
		updates++
		return event.Validate()
	}})
	var summary struct {
		Imported int `json:"imported"`
	}
	if err != nil || json.Unmarshal(result.Data, &summary) != nil || summary.Imported != largeLegacyLogs || updates < 2 {
		t.Fatalf("native import progress=%d result=%s err=%v", updates, result.Data, err)
	}
	api.mu.Lock()
	actual, logs := api.digest.Sum(nil), api.logs
	api.mu.Unlock()
	if logs != largeLegacyLogs || !bytes.Equal(actual, expected) {
		t.Fatalf("native import changed records: logs=%d digest=%x expected=%x", logs, actual, expected)
	}
}
