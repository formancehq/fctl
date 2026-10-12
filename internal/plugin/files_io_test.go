package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type pluginInputCase struct{ name, format, input, want string }

func TestPluginInputFormats(t *testing.T) {
	t.Parallel()
	for _, tc := range []pluginInputCase{
		{"numscript", "string", "send [USD 10] (\n source = @world\n destination = @alice\n)\n", ""},
		{"ndjson", "string", "{\"amount\":90071992547409931234567890}\n{\"amount\":1e+19}\n", ""},
		{"json", "json", `{"amount":90071992547409931234567890}`, `{"amount":90071992547409931234567890}`},
		{"default json", "", `[1,2]`, `[1,2]`},
		{"yaml", "yaml", "name: schema\namount: 9007199254740993\n", `{"amount":9007199254740993,"name":"schema"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, source := range []string{"file", "stdin"} {
				testPluginInputFormat(t, tc, source)
			}
		})
	}
}

func pluginInputCommand(t *testing.T, input, source string) (*cobra.Command, string) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetIn(strings.NewReader(input))
	if source == "stdin" {
		return cmd, "-"
	}
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	return cmd, path
}

func testPluginInputFormat(t *testing.T, tc pluginInputCase, source string) {
	t.Helper()
	cmd, path := pluginInputCommand(t, tc.input, source)
	request := pluginsdk.ExecuteRequest{Args: []string{"ledger", path}}
	got, err := ReadPluginInput(cmd, &pluginsdk.FileSpec{ReadArgument: new(1), ReadFormat: tc.format}, request)
	if err != nil {
		t.Fatal(err)
	}
	if tc.format == "string" {
		var decoded string
		if err := json.Unmarshal(got.Body, &decoded); err != nil || decoded != tc.input {
			t.Fatalf("text changed: %s (%v)", got.Body, err)
		}
	} else if string(got.Body) != tc.want {
		t.Fatalf("body = %s, want %s", got.Body, tc.want)
	}
	if got.Args[1] != path || request.Body != nil {
		t.Fatal("host changed the source request")
	}
}

func TestPluginInputBoundsAndValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, format, input string
	}{
		{"oversized", "string", strings.Repeat("x", maxPluginInput+1)},
		{"encoded bound", "string", strings.Repeat("\n", maxPluginInput/2+1)},
		{"invalid JSON", "json", "{"},
		{"multiple JSON", "json", "{} {}"},
		{"duplicate YAML", "yaml", "name: one\nname: two\n"},
		{"invalid YAML", "yaml", "name: ["},
		{"unsupported format", "xml", "{}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, source := range []string{"file", "stdin"} {
				cmd, path := pluginInputCommand(t, tc.input, source)
				_, err := ReadPluginInput(cmd, &pluginsdk.FileSpec{ReadFlag: "file", ReadFormat: tc.format}, pluginsdk.ExecuteRequest{Flags: map[string]string{"file": path}})
				if err == nil {
					t.Fatal("invalid file input accepted")
				}
			}
		})
	}
}

func TestPluginLocalInputAndExactLimit(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	for _, path := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		if _, err := ReadPluginInput(cmd, &pluginsdk.FileSpec{ReadArgument: new(0)}, pluginsdk.ExecuteRequest{Args: []string{path}}); err == nil {
			t.Fatal("non-file source accepted")
		}
	}
	// Exactly 4 MiB of valid JSON is accepted without decoding its numbers.
	raw := []byte(`{"amount":90071992547409931234567890}`)
	raw = append(raw, bytes.Repeat([]byte(" "), maxPluginInput-len(raw))...)
	got, err := decodePluginInput("json", raw)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("input at the limit changed: %v", err)
	}
}

func TestPluginStdinCancellationKeepsInputOpen(t *testing.T) {
	t.Parallel()
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
		if err := output.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		_, err := readPluginStdin(ctx, input)
		finished <- err
	}()
	<-started
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stdin read did not stop on cancellation")
	}
	if _, err := output.Write([]byte("still open")); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, len("still open"))
	if _, err := io.ReadFull(input, data); err != nil || string(data) != "still open" {
		t.Fatalf("stdin was closed or consumed: %q (%v)", data, err)
	}
}

func TestPluginNDJSONRoundTrip(t *testing.T) {
	t.Parallel()
	array := `[{"amount":90071992547409931234567890,"ratio":1.234567890123456789e+30}, {"amount":18446744073709551616}]`
	want := "{\"amount\":90071992547409931234567890,\"ratio\":1.234567890123456789e+30}\n{\"amount\":18446744073709551616}\n"
	for _, data := range []string{array, `{"logs":` + array + `}`, `{"data":` + array + `}`} {
		var output bytes.Buffer
		if err := writePluginNDJSON(&output, json.RawMessage(data)); err != nil {
			t.Fatal(err)
		}
		if output.String() != want {
			t.Fatalf("NDJSON changed: %s", output.Bytes())
		}
		body, err := decodePluginInput("string", output.Bytes())
		var input string
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &input); err != nil || input != want {
			t.Fatalf("import changed exported logs: %v", err)
		}
	}
}

func TestPluginInvalidNDJSONOutput(t *testing.T) {
	t.Parallel()
	for _, data := range []string{`null`, `{}`, `{"logs":{}}`, `[`, `1`} {
		var output bytes.Buffer
		if err := writePluginNDJSON(&output, json.RawMessage(data)); err == nil || output.Len() != 0 {
			t.Fatalf("invalid export %q wrote output or succeeded", data)
		}
	}
	var empty bytes.Buffer
	if err := writePluginNDJSON(&empty, json.RawMessage(`[]`)); err != nil || empty.Len() != 0 {
		t.Fatalf("empty export failed: %v", err)
	}
}

func TestPluginYAMLOutput(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	raw := json.RawMessage(`{"name":"schema","amount":9007199254740993,"properties":{"id":{"type":"string"}}}`)
	if err := writePluginOutput(&output, "yaml", raw); err != nil {
		t.Fatal(err)
	}
	converted, err := yaml.YAMLToJSON(output.Bytes())
	if err != nil || !bytes.Contains(converted, []byte(`9007199254740993`)) || !bytes.Contains(output.Bytes(), []byte("name: schema")) {
		t.Fatalf("YAML output = %s (%v)", output.Bytes(), err)
	}
}

func TestPluginAtomicWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "export.ndjson")
	if err := os.WriteFile(path, []byte("old content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil { //nolint:gosec // Seed permissive existing output to verify the atomic replacement uses private permissions.
		t.Fatal(err)
	}
	for range 2 {
		if err := writePluginFile(t.Context(), path, []byte("new content\n")); err != nil {
			t.Fatal(err)
		}
		assertPrivatePluginExport(t, path)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := writePluginFile(ctx, path, []byte("canceled")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled write = %v", err)
	}
	if err := writePluginFile(t.Context(), dir, []byte("rename fails")); err == nil {
		t.Fatal("export replaced a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "export.ndjson" {
		t.Fatalf("temporary exports leaked: %v (%v)", entries, err)
	}
	assertPrivatePluginExport(t, path)
}

func assertPrivatePluginExport(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // The path is the test-owned temporary export file.
	if err != nil || string(data) != "new content\n" {
		t.Fatal("failed write replaced previous export")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("export permission: %v (%v)", info, err)
	}
}
