package plugin_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/interactive"
	"github.com/formancehq/fctl/v4/internal/plugin"
)

type fakeStreamingPlugin struct {
	fakePlugin
	stream func(context.Context, pluginsdk.ExecuteRequest, pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error)
}

func (p *fakeStreamingPlugin) ExecuteStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
	return p.stream(ctx, request, streams)
}

func streamManifest(input, output bool) pluginsdk.Manifest {
	manifest := fileManifest()
	spec := &manifest.Root.Subcommands[0]
	spec.Streams = &pluginsdk.StreamSpec{Input: input, Output: output}
	if output {
		spec.Files.WriteFormat = "ndjson"
		spec.Flags[3].Default = "ndjson"
	}
	return manifest
}

func streamAdapter(t *testing.T, manifest pluginsdk.Manifest, execute func(context.Context, pluginsdk.ExecuteRequest, pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error)) *cobra.Command {
	t.Helper()
	return attach(t, manifest, func(context.Context, string) (*api.Client, error) {
		return api.New("https://ledger.example", &http.Client{})
	}, func(*http.Client) pluginsdk.Plugin {
		return &fakeStreamingPlugin{fakePlugin: fakePlugin{execute: func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			t.Fatal("declared streams used unary execution")
			return pluginsdk.ExecuteResponse{}, nil
		}}, stream: execute}
	})
}

func largeStreamFile(t *testing.T) (string, int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.ndjson")
	file, err := os.Create(path) //nolint:gosec // Test-owned temporary source.
	if err != nil {
		t.Fatal(err)
	}
	piece := bytes.Repeat([]byte("{\"n\":1}\n"), 8192)
	const size = 65 << 20
	for range size / len(piece) {
		if _, err := file.Write(piece); err != nil {
			t.Fatal(errors.Join(err, file.Close()))
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path, size
}

func streamFileHash(t *testing.T, path string) []byte {
	t.Helper()
	file, err := os.Open(path) //nolint:gosec // Test-owned temporary file.
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err := errors.Join(err, file.Close()); err != nil {
		t.Fatal(err)
	}
	return hash.Sum(nil)
}

func assertNoStreamTemp(t *testing.T, path string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".fctl-export-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary export remains: %v (%v)", files, err)
	}
}

func TestHostStreamsLargeFileInputAndAtomicOutput(t *testing.T) {
	t.Parallel()
	source, size := largeStreamFile(t)
	destination := filepath.Join(t.TempDir(), "export.ndjson")
	if err := os.WriteFile(destination, []byte("previous\n"), 0o644); err != nil { //nolint:gosec // Verify a replacement becomes private even when the original file was not.
		t.Fatal(err)
	}
	manifest := streamManifest(true, true)
	manifest.Root.Subcommands[0].Flags[0].Required = true
	root := streamAdapter(t, manifest, func(_ context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		if request.Body != nil || streams.Input == nil || streams.Output == nil || streams.InputSize == nil || *streams.InputSize != size {
			t.Fatal("large source was buffered or its size was lost")
		}
		n, err := io.CopyBuffer(streams.Output, streams.Input, make([]byte, 64<<10))
		if n != size {
			t.Fatalf("stream copied %d, want %d", n, size)
		}
		return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"exported":true}`)}, err
	})
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"ledger", "invoke", source, "--file=" + destination})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Size() != size || info.Mode().Perm() != 0o600 {
		t.Fatalf("atomic private export failed: %v (%v)", info, err)
	}
	if !bytes.Equal(streamFileHash(t, source), streamFileHash(t, destination)) || !strings.Contains(stdout.String(), `"exported": true`) {
		t.Fatal("export bytes or final file summary changed")
	}
	assertNoStreamTemp(t, destination)
}

func TestHostStreamFailurePreservesDestination(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"plugin", "write", "cancel", "invalid report"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			testHostStreamFailure(t, failure)
		})
	}
}

func testHostStreamFailure(t *testing.T, failure string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "export.ndjson")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	root := streamAdapter(t, streamManifest(false, true), func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		if _, err := io.WriteString(streams.Output, "{\"partial\":true}\n"); err != nil {
			return pluginsdk.ExecuteResponse{}, err
		}
		return pluginsdk.ExecuteResponse{}, streamFailure(t, failure, cancel, streams)
	})
	root.SetOut(io.Discard)
	root.SetArgs([]string{"ledger", "invoke", "--file=" + path})
	err := root.ExecuteContext(ctx)
	if err == nil || (failure == "cancel" && !errors.Is(err, context.Canceled)) {
		t.Fatalf("stream failure not returned: %v", err)
	}
	data, readErr := os.ReadFile(path) //nolint:gosec // Test-owned temporary destination.
	if readErr != nil || string(data) != "original\n" {
		t.Fatalf("failed export replaced destination: %q (%v)", data, readErr)
	}
	assertNoStreamTemp(t, path)
}

func streamFailure(t *testing.T, failure string, cancel context.CancelFunc, streams pluginsdk.ExecutionStreams) error {
	t.Helper()
	switch failure {
	case "cancel":
		cancel()
		return nil // Ignored cancellation must still prevent commit.
	case "invalid report":
		if err := streams.Report(pluginsdk.ProgressEvent{Phase: "bad\x1b", Unit: "records"}); err == nil {
			t.Fatal("invalid hidden progress accepted")
		}
		return nil // Ignored Report errors must still prevent commit.
	case "write":
		return io.ErrShortWrite
	default:
		return errors.New("plugin failed")
	}
}

func TestHostStreamStdoutNeverAppendsSummary(t *testing.T) {
	t.Parallel()
	for _, destination := range []string{"", "-"} {
		for _, failed := range []bool{false, true} {
			root := streamAdapter(t, streamManifest(false, true), func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
				_, err := io.WriteString(streams.Output, "{\"record\":1}\n")
				if failed {
					err = errors.Join(err, errors.New("partial export"))
				}
				return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"summary":"private"}`)}, err
			})
			var out bytes.Buffer
			root.SetOut(&out)
			args := []string{"ledger", "invoke"}
			if destination != "" {
				args = append(args, "--file="+destination)
			}
			root.SetArgs(args)
			err := root.ExecuteContext(t.Context())
			if (err != nil) != failed || out.String() != "{\"record\":1}\n" {
				t.Fatalf("stdout export corrupted: %q (%v)", out.String(), err)
			}
		}
	}
}

func TestHostStreamStdinAndLegacyBody(t *testing.T) {
	t.Parallel()
	for _, body := range []bool{false, true} {
		called := false
		root := streamAdapter(t, streamManifest(true, false), func(_ context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
			called = true
			assertStreamImportInput(t, body, request, streams)
			return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"imported":1}`)}, nil
		})
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		args := []string{"ledger", "invoke", "-"}
		root.SetIn(strings.NewReader("{\"raw\":true}\n"))
		if body {
			root.SetIn(forbiddenReader{t})
			args = []string{"ledger", "invoke", `--data={"legacy":true}`}
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(t.Context()); err != nil || !called || !strings.Contains(stdout.String(), `"imported": 1`) {
			t.Fatalf("import summary failed: %q (%v)", stdout.String(), err)
		}
	}
}

func assertStreamImportInput(t *testing.T, body bool, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) {
	t.Helper()
	if streams.Output != nil || streams.InputSize != nil {
		t.Fatal("import received output or invented stdin size")
	}
	if body {
		if streams.Input != nil || string(request.Body) != `{"legacy":true}` {
			t.Fatal("legacy body became a raw stream")
		}
		return
	}
	data, err := io.ReadAll(streams.Input)
	if err != nil || string(data) != "{\"raw\":true}\n" || request.Body != nil {
		t.Fatalf("stdin was changed or buffered: %q (%v)", data, err)
	}
}

func TestHostStreamFormatAndSourceConflictsBeforeIO(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-", "--format=yaml"}, {"-", "--data={}"}, {"-", "--input=-"}} {
		root := attach(t, streamManifest(true, true), func(context.Context, string) (*api.Client, error) {
			t.Fatal("invalid stream request reached auth")
			return nil, nil
		}, inertFactory)
		root.SetIn(forbiddenReader{t})
		root.SetOut(io.Discard)
		root.SetArgs(append([]string{"ledger", "invoke"}, args...))
		if err := root.ExecuteContext(t.Context()); err == nil {
			t.Fatal("invalid stream request accepted")
		}
	}
}

type streamConfirmationRunner struct {
	t        *testing.T
	accepted bool
}

func (r streamConfirmationRunner) Run(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
	if len(fields) != 1 || fields[0].Kind != "confirm" {
		r.t.Fatal("stream input prompted for a body instead of confirmation")
	}
	if r.accepted {
		return []string{"true"}, nil
	}
	return []string{"false"}, nil
}

func TestHostStreamInteractiveConfirmationPrecedesFileOpen(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	for _, accepted := range []bool{false, true} {
		manifest := streamManifest(true, true)
		spec := &manifest.Root.Subcommands[0]
		spec.Confirm = true
		spec.Flags[0].Required = true
		spec.Inputs = []pluginsdk.InputSpec{{Title: "Body", Kind: "text", Flag: "data", ValueType: "json", Required: true}}
		called := false
		root := streamAdapter(t, manifest, func(_ context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
			called = true
			if request.Flags["confirm"] != "true" || request.Body != nil {
				t.Fatal("unconfirmed or buffered stream executed")
			}
			_, err := io.Copy(streams.Output, streams.Input)
			return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"summary":true}`)}, err
		})
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"ledger", "invoke", "-"})
		root.SetIn(forbiddenReader{t})
		if accepted {
			root.SetIn(strings.NewReader("{\"record\":1}\n"))
		}
		ctx := interactive.WithRunner(t.Context(), streamConfirmationRunner{t, accepted})
		err := root.ExecuteContext(ctx)
		if accepted {
			if err != nil || !called || out.String() != "{\"record\":1}\n" {
				t.Fatalf("interactive stream failed: %q (%v)", out.String(), err)
			}
		} else if !errors.Is(err, interactive.ErrCanceled) || called || out.Len() != 0 {
			t.Fatalf("declined stream executed: %v", err)
		}
	}
}

func TestHostStreamStdinCancellation(t *testing.T) {
	t.Parallel()
	input, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := errors.Join(input.Close(), writer.Close()); err != nil {
			t.Error(err)
		}
	})
	started := make(chan struct{})
	root := streamAdapter(t, streamManifest(true, false), func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		close(started)
		_, err := io.Copy(io.Discard, streams.Input)
		return pluginsdk.ExecuteResponse{}, err
	})
	root.SetIn(input)
	root.SetOut(io.Discard)
	root.SetArgs([]string{"ledger", "invoke", "-"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("stdin was buffered before execution")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("stdin cancellation lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stdin read did not unblock on cancellation")
	}
}

func TestHostStreamRegistryDeclarations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		valid bool
		edit  func(*pluginsdk.CommandSpec)
	}{
		{"raw string", true, func(*pluginsdk.CommandSpec) {}},
		{"raw ndjson", true, func(s *pluginsdk.CommandSpec) { s.Files.ReadFormat = "ndjson" }},
		{"empty stream declaration", true, func(s *pluginsdk.CommandSpec) { s.Streams = &pluginsdk.StreamSpec{}; s.Files = nil }},
		{"missing files", false, func(s *pluginsdk.CommandSpec) { s.Files = nil }},
		{"missing source", false, func(s *pluginsdk.CommandSpec) { s.Files.ReadArgument = nil; s.Files.ReadFlag = "" }},
		{"YAML conversion", false, func(s *pluginsdk.CommandSpec) { s.Files.ReadFormat = "yaml" }},
		{"JSON conversion", false, func(s *pluginsdk.CommandSpec) { s.Files.ReadFormat = "json" }},
		{"mismatched output default", false, func(s *pluginsdk.CommandSpec) { s.Flags[3].Default = "json" }},
		{"group", false, func(s *pluginsdk.CommandSpec) { s.Runnable = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := streamManifest(true, true)
			tc.edit(&manifest.Root.Subcommands[0])
			registry := new(plugin.Registry)
			err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, inertFactory)
			if (err == nil) != tc.valid {
				t.Fatalf("declaration valid=%t, error=%v", tc.valid, err)
			}
		})
	}
}

func TestHostStreamProgressOnly(t *testing.T) {
	t.Parallel()
	manifest := streamManifest(false, false)
	manifest.Root.Subcommands[0].Files = nil
	root := streamAdapter(t, manifest, func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		if streams.Input != nil || streams.Output != nil || streams.InputSize != nil || streams.Report == nil {
			t.Fatal("progress-only command acquired raw streams")
		}
		err := streams.Report(pluginsdk.ProgressEvent{Phase: "apply", Completed: 1, Total: new(int64(1)), Unit: "records"})
		return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"ok":true}`)}, err
	})
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"ledger", "invoke"})
	if err := root.ExecuteContext(t.Context()); err != nil || !strings.Contains(stdout.String(), `"ok": true`) || stderr.Len() != 0 {
		t.Fatalf("progress-only execution failed: out=%q, err=%q (%v)", stdout.String(), stderr.String(), err)
	}
}

type streamBrokenWriter struct{ err error }

func (w streamBrokenWriter) Write([]byte) (int, error) { return 0, w.err }

func TestHostStreamIgnoredStdoutWriteFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("broken stdout")
	root := streamAdapter(t, streamManifest(false, true), func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		if _, err := streams.Output.Write([]byte("record\n")); !errors.Is(err, failure) {
			t.Fatalf("writer lost failure: %v", err)
		}
		return pluginsdk.ExecuteResponse{Data: json.RawMessage(`{"summary":true}`)}, nil
	})
	root.SetOut(streamBrokenWriter{failure})
	root.SetArgs([]string{"ledger", "invoke"})
	if err := root.ExecuteContext(t.Context()); !errors.Is(err, failure) {
		t.Fatalf("host ignored export write error: %v", err)
	}
}

func TestHostStreamUnsupportedPluginBeforeSourceRead(t *testing.T) {
	t.Parallel()
	root := fileAdapter(t, streamManifest(true, false), func(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
		t.Fatal("unsupported stream fell back to unary")
		return pluginsdk.ExecuteResponse{}, nil
	})
	root.SetIn(forbiddenReader{t})
	root.SetArgs([]string{"ledger", "invoke", "-"})
	if err := root.ExecuteContext(t.Context()); err == nil || !strings.Contains(err.Error(), "does not support") {
		t.Fatalf("unsupported streaming plugin accepted: %v", err)
	}
}

type streamBlockedOutput struct {
	*io.PipeWriter
	started chan struct{}
}

func (w *streamBlockedOutput) Write(data []byte) (int, error) {
	close(w.started)
	return w.PipeWriter.Write(data)
}

func TestHostStreamStdoutCancellation(t *testing.T) {
	t.Parallel()
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		if err := errors.Join(reader.Close(), writer.Close()); err != nil {
			t.Error(err)
		}
	})
	output := &streamBlockedOutput{PipeWriter: writer, started: make(chan struct{})}
	root := streamAdapter(t, streamManifest(false, true), func(_ context.Context, _ pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
		_, err := io.WriteString(streams.Output, "record\n")
		return pluginsdk.ExecuteResponse{}, err
	})
	root.SetOut(output)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		return command.ConfigureOutput(cmd, "json", "never")
	}
	root.SetArgs([]string{"ledger", "invoke"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()
	select {
	case <-output.started:
	case <-time.After(5 * time.Second):
		t.Fatal("export was not started")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || err.Error() != context.Canceled.Error() {
			t.Fatalf("stdout cancellation was lost or repeated: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stdout Write did not unblock on cancellation")
	}
}
