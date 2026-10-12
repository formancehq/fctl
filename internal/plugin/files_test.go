package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/interactive"
	"github.com/formancehq/fctl/v4/internal/plugin"
)

func fileManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "ledger", Version: "test", Service: "ledger", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{
		Use: "ledger", Subcommands: []pluginsdk.CommandSpec{{
			Use: "invoke [FILE]", Runnable: true, Args: pluginsdk.ArgsSpec{Max: 1}, Files: &pluginsdk.FileSpec{
				ReadArgument: new(0), ReadFlag: "input", ReadFormat: "string", WriteFlag: "file", FormatFlag: "format",
			}, Flags: []pluginsdk.FlagSpec{
				{Name: "data", Type: "string", Body: true}, {Name: "input", Type: "string"},
				{Name: "file", Type: "string"}, {Name: "format", Type: "string", Default: "json"},
				{Name: "confirm", Type: "bool", Default: "false"}, {Name: "experimental", Type: "bool", Default: "false"},
			},
		}},
	}}
}

func fileAdapter(t *testing.T, manifest pluginsdk.Manifest, execute func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error)) *cobra.Command {
	t.Helper()
	return attach(t, manifest, func(context.Context, string) (*api.Client, error) {
		return api.New("https://ledger.example", &http.Client{})
	}, func(*http.Client) pluginsdk.Plugin { return &fakePlugin{execute: execute} })
}

func TestFileAdapterInputAndRequiredBody(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-"}, {"--input=-"}} {
		m := fileManifest()
		m.Root.Subcommands[0].Flags[0].Required = true
		root := fileAdapter(t, m, func(_ context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			var script string
			if err := json.Unmarshal(request.Body, &script); err != nil || script != "send [USD 10]" {
				t.Fatalf("plugin did not receive Numscript: %s (%v)", request.Body, err)
			}
			if _, err := pluginsdk.NormalizeRequest(m, request); err != nil {
				t.Fatal(err)
			}
			return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"ok":true}`)}, nil
		})
		root.SetIn(strings.NewReader("send [USD 10]"))
		root.SetOut(io.Discard)
		root.SetArgs(append([]string{"ledger", "invoke"}, args...))
		if err := root.ExecuteContext(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFileAdapterGatesAndConflictsBeforeIO(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
		edit func(*pluginsdk.CommandSpec)
	}{
		{"confirmation", []string{"-"}, func(s *pluginsdk.CommandSpec) { s.Confirm = true }},
		{"SDK gate", []string{"-"}, func(s *pluginsdk.CommandSpec) { s.Flags[5].RequireTrue = true }},
		{"data conflict", []string{"-", "--data=-"}, nil},
		{"empty data conflict", []string{"-", "--data="}, nil},
		{"sources conflict", []string{"-", "--input=-"}, nil},
		{"empty input", []string{"--input="}, nil},
		{"empty output", []string{"--file="}, nil},
		{"invalid format", []string{"-", "--format=xml"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := fileManifest()
			if tc.edit != nil {
				tc.edit(&m.Root.Subcommands[0])
			}
			root := attach(t, m, func(context.Context, string) (*api.Client, error) {
				t.Fatal("invalid file command resolved client")
				return nil, nil
			}, func(*http.Client) pluginsdk.Plugin {
				t.Fatal("invalid file command constructed plugin")
				return nil
			})
			root.SetIn(forbiddenReader{t})
			root.SetOut(io.Discard)
			root.SetArgs(append([]string{"ledger", "invoke"}, tc.args...))
			if err := root.ExecuteContext(t.Context()); err == nil {
				t.Fatal("invalid command accepted")
			}
		})
	}
}

func TestFileAdapterOptionalSourceAndExplicitBody(t *testing.T) {
	t.Parallel()
	for _, body := range []string{"", `{"amount":90071992547409931234567890}`} {
		called := false
		root := fileAdapter(t, fileManifest(), func(_ context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			called = true
			if string(request.Body) != body {
				t.Fatalf("body = %s, want %s", request.Body, body)
			}
			return pluginsdk.ExecuteResponse{}, nil
		})
		root.SetIn(forbiddenReader{t})
		args := []string{"ledger", "invoke"}
		if body != "" {
			args = append(args, "--data="+body)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(t.Context()); err != nil || !called {
			t.Fatalf("optional source command failed: %v", err)
		}
	}
}

func TestFileAdapterExport(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"json", "ndjson", "yaml", "yml"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "export")
			root := fileAdapter(t, fileManifest(), func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
				return pluginsdk.ExecuteResponse{Data: json.RawMessage(`[{"amount":9007199254740993}]`)}, nil
			})
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetArgs([]string{"ledger", "invoke", "--file=" + path, "--format=" + format})
			if err := root.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path) //nolint:gosec // The path is the test-owned temporary export file.
			if err != nil || !bytes.Contains(data, []byte("9007199254740993")) || output.Len() != 0 {
				t.Fatalf("export failed: %q, stdout=%q (%v)", data, output.String(), err)
			}
		})
	}
	for _, args := range [][]string{{"--format=ndjson"}, {"--format=ndjson", "--file=-"}} {
		root := fileAdapter(t, fileManifest(), func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"logs":[{"amount":90071992547409931234567890}]}`)}, nil
		})
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetArgs(append([]string{"ledger", "invoke"}, args...))
		if err := root.ExecuteContext(t.Context()); err != nil || output.String() != "{\"amount\":90071992547409931234567890}\n" {
			t.Fatalf("stdout export = %q (%v)", output.String(), err)
		}
	}
}

type pluginExportErrorCase struct {
	name, data string
	failed     bool
	cancel     bool
}

func TestFileAdapterNeverExportsErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []pluginExportErrorCase{
		{"partial failure", `[{"id":1}]`, true, false},
		{"invalid response", `{`, false, false},
		{"wrong export shape", `{}`, false, false},
		{"canceled execution", `[{"id":1}]`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testFileAdapterExportError(t, tc)
		})
	}
}

func testFileAdapterExportError(t *testing.T, tc pluginExportErrorCase) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "export")
	if err := os.WriteFile(path, []byte("previous export"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	execErr := errors.New("plugin failed")
	root := fileAdapter(t, fileManifest(), func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
		if tc.cancel {
			cancel()
		}
		response := pluginsdk.ExecuteResponse{Data: json.RawMessage(tc.data)}
		if tc.failed {
			return response, execErr
		}
		return response, nil
	})
	root.SetOut(io.Discard)
	root.SetArgs([]string{"ledger", "invoke", "--file=" + path, "--format=ndjson"})
	err := root.ExecuteContext(ctx)
	if err == nil || (tc.failed && !errors.Is(err, execErr)) || (tc.cancel && !errors.Is(err, context.Canceled)) {
		t.Fatalf("export lost error: %v", err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // The path is the test-owned temporary export file.
	if err != nil || string(data) != "previous export" {
		t.Fatalf("failed export replaced destination: %q (%v)", data, err)
	}
}

type fileConfirmRunner struct{ accepted bool }

func (r fileConfirmRunner) Run(context.Context, *cobra.Command, []interactive.Field) ([]string, error) {
	if r.accepted {
		return []string{"true"}, nil
	}
	return []string{"false"}, nil
}

func TestInteractiveFileConfirmationBeforeIO(t *testing.T) {
	// Interactive enablement checks process-level environment settings.
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	for _, accepted := range []bool{false, true} {
		testInteractiveFileConfirmation(t, accepted)
	}
}

func testInteractiveFileConfirmation(t *testing.T, accepted bool) {
	t.Helper()
	m := fileManifest()
	m.Root.Subcommands[0].Confirm = true
	m.Root.Subcommands[0].Flags[0].Required = true
	m.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{Title: "Script", Kind: "text", BodyPointer: "/script", Required: true}}
	path := filepath.Join(t.TempDir(), "export")
	called := false
	root := fileAdapter(t, m, func(_ context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
		called = true
		if string(request.Body) != `"send [USD 10]"` || request.Flags["confirm"] != "true" {
			t.Fatalf("incorrect interactive request: %+v", request)
		}
		return pluginsdk.ExecuteResponse{Data: json.RawMessage(`[]`)}, nil
	})
	root.SetOut(io.Discard)
	root.SetIn(forbiddenReader{t})
	if accepted {
		root.SetIn(strings.NewReader("send [USD 10]"))
	}
	root.SetArgs([]string{"ledger", "invoke", "-", "--file=" + path, "--format=ndjson"})
	err := root.ExecuteContext(interactive.WithRunner(t.Context(), fileConfirmRunner{accepted}))
	if !accepted {
		if !errors.Is(err, interactive.ErrCanceled) || called {
			t.Fatalf("declined confirmation = %v, executed=%v", err, called)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("declined confirmation created a file")
		}
	} else if err != nil || !called {
		t.Fatalf("accepted confirmation failed: %v", err)
	}
}

func TestFileMetadataValidationAndClone(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		edit func(*pluginsdk.Manifest)
	}{
		{"group", func(m *pluginsdk.Manifest) { m.Root.Files = m.Root.Subcommands[0].Files }},
		{"runnable group", func(m *pluginsdk.Manifest) {
			m.Root.Runnable = true
			m.Root.Files = &pluginsdk.FileSpec{WriteFormat: "yaml"}
		}},
		{"negative index", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.ReadArgument = new(-1) }},
		{"out of bounds", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.ReadArgument = new(1) }},
		{"read format", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.ReadFormat = "xml" }},
		{"write format", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.WriteFormat = "xml" }},
		{"missing read flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.ReadFlag = "missing" }},
		{"non-string flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.WriteFlag = "confirm" }},
		{"body flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.ReadFlag = "data" }},
		{"same file flags", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.WriteFlag = "input" }},
		{"same format flag", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Files.FormatFlag = "input" }},
		{"bad format default", func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Flags[3].Default = "xml" }},
		{"format without source", func(m *pluginsdk.Manifest) {
			m.Root.Subcommands[0].Files.ReadArgument = nil
			m.Root.Subcommands[0].Files.ReadFlag = ""
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := fileManifest()
			tc.edit(&m)
			registry := &plugin.Registry{}
			if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, inertFactory); err == nil {
				t.Fatal("invalid file metadata accepted")
			}
		})
	}
	m := fileManifest()
	// Persistent string flags can be bound by a leaf. Optional source args
	// remain optional, and output-only metadata can format stdout.
	m.Root.Flags = []pluginsdk.FlagSpec{{Name: "output-file", Type: "string", Persistent: true}}
	m.Root.Subcommands[0].Files.WriteFlag = "output-file"
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: m}, inertFactory); err != nil {
		t.Fatal(err)
	}
	*m.Root.Subcommands[0].Files.ReadArgument = 10
	m.Root.Subcommands[0].Files.WriteFormat = "invalid"
	listed := registry.List()
	listed[0].Root.Subcommands[0].Files.ReadArgument = new(20)
	listed[0].Root.Subcommands[0].Files.WriteFlag = "changed"
	again := registry.List()[0].Root.Subcommands[0].Files
	if *again.ReadArgument != 0 || again.WriteFlag != "output-file" || again.WriteFormat != "" {
		t.Fatal("file metadata was not deeply cloned")
	}
}
