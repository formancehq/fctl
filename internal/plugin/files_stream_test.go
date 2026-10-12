package plugin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type pluginStreamZeros struct{}

func (pluginStreamZeros) Read(data []byte) (int, error) {
	clear(data)
	return len(data), nil
}

type pluginStreamTestBody struct {
	io.Reader
	closed bool
}

func (b *pluginStreamTestBody) Close() error {
	b.closed = true
	return nil
}

func TestPluginStreamURLLargeBodyAndSize(t *testing.T) {
	t.Parallel()
	for _, known := range []bool{false, true} {
		const size int64 = 65 << 20
		body := &pluginStreamTestBody{Reader: io.LimitReader(pluginStreamZeros{}, size)}
		client := newPluginURLClient()
		calls := 0
		client.Transport = pluginURLTestTransport(func(request *http.Request) (*http.Response, error) {
			calls++
			if request.Method != http.MethodGet || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
				t.Fatal("source fetch inherited credentials or changed method")
			}
			length := int64(-1)
			if known {
				length = size
			}
			return &http.Response{StatusCode: 200, Body: body, ContentLength: length, Request: request}, nil
		})
		input, gotSize, err := openPluginStreamURL(t.Context(), "https://example.test/input", client)
		if err != nil || (gotSize != nil) != known || (gotSize != nil && *gotSize != size) {
			t.Fatalf("URL size lost: %v (%v)", gotSize, err)
		}
		n, err := io.Copy(io.Discard, input)
		if err := errors.Join(err, input.Close()); err != nil || n != size || !body.closed || calls != 1 {
			t.Fatalf("unbounded URL input failed: bytes=%d, calls=%d, closed=%t (%v)", n, calls, body.closed, err)
		}
	}
}

func TestPluginStreamURLPolicyAndNoRetries(t *testing.T) {
	t.Parallel()
	client := newPluginURLClient()
	calls := 0
	client.Transport = pluginURLTestTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("fetch failed")
	})
	for _, source := range []string{
		"http://example.test/input", "https:opaque", "https:///input", "https://%zz/input",
		"https://private-user:private-pass@example.test/input", "https://example.test/input#private-fragment", "https://example.test/input#",
	} {
		if _, _, err := openPluginStreamURL(t.Context(), source, client); err == nil || strings.Contains(err.Error(), "private-") {
			t.Fatalf("invalid URL accepted or exposed: %v", err)
		}
	}
	if calls != 0 {
		t.Fatal("invalid URL reached transport")
	}
	if _, _, err := openPluginStreamURL(t.Context(), "https://example.test/input?private-secret", client); err == nil || strings.Contains(err.Error(), "private-secret") || calls != 1 {
		t.Fatalf("failed URL exposed query or retried: calls=%d (%v)", calls, err)
	}
}

func TestPluginStreamURLRedirectAndCancellation(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.URL.Path == "/redirect" {
			writer.Header().Set("Location", "http://example.test/private")
			writer.WriteHeader(http.StatusFound)
			return
		}
		writer.WriteHeader(http.StatusOK)
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()
	client := pluginURLTestClient(t, server)
	if _, _, err := openPluginStreamURL(t.Context(), server.URL+"/redirect", client); err == nil || calls.Load() != 1 {
		t.Fatalf("stream followed insecure redirect: calls=%d (%v)", calls.Load(), err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	input, _, err := openPluginStreamURL(ctx, server.URL+"/blocking", client)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	}()
	<-started
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, &pluginStreamReader{ctx: ctx, reader: input, failures: &pluginStreamErrors{}})
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("HTTP stream lost cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP read stayed blocked")
	}
	if calls.Load() != 2 {
		t.Fatal("host retried source HTTP")
	}
}

func TestPluginStreamRegularStdinRemainsOpen(t *testing.T) {
	t.Parallel()
	file := regularStdinFile(t, "{\"record\":1}\n")
	input, size, err := openPluginStreamStdin(t.Context(), file)
	if err != nil || size != nil {
		t.Fatalf("regular stdin opening failed: %v", err)
	}
	data, err := io.ReadAll(input)
	if err := errors.Join(err, input.Close()); err != nil || string(data) != "{\"record\":1}\n" {
		t.Fatalf("regular stdin changed: %q (%v)", data, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("host closed stdin: %v", err)
	}
}

func TestPluginStreamReaderCloseInterruptsPendingStdin(t *testing.T) {
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
	source, _, err := openPluginStreamStdin(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	var reader io.ReadCloser = &pluginStreamReader{ctx: t.Context(), reader: source, failures: &pluginStreamErrors{}}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, err := io.Copy(io.Discard, reader)
		done <- err
	}()
	<-started
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closing the interruptible input did not abort its read")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("wrapper Close left a blocked reader")
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("source Close was not idempotent: %v", err)
	}
	if _, err := input.Stat(); err != nil {
		t.Fatalf("interrupt closed the caller's stdin: %v", err)
	}
}

type pluginStreamShortWriter struct{}

func (pluginStreamShortWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func TestPluginStreamOutputWriteAndCommitErrors(t *testing.T) {
	t.Parallel()
	var failures pluginStreamErrors
	writer := &pluginStreamWriter{ctx: t.Context(), writer: pluginStreamShortWriter{}, failures: &failures}
	if _, err := writer.Write([]byte("record")); !errors.Is(err, io.ErrShortWrite) || !errors.Is(failures.Err(), io.ErrShortWrite) {
		t.Fatalf("short write was ignored: %v", err)
	}
	path := filepath.Join(t.TempDir(), "destination")
	if err := os.WriteFile(path, []byte("previous\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	spec := pluginsdk.CommandSpec{Streams: &pluginsdk.StreamSpec{Output: true}, Files: &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}}
	output, err := openPluginStreamOutput(cmd, spec, pluginsdk.ExecuteRequest{Flags: map[string]string{"file": path}}, &pluginStreamErrors{})
	if err != nil {
		t.Fatal(err)
	}
	// A real file write failure and failed sync must not publish the temp file.
	if err := output.file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := output.writer.Write([]byte("replacement\n")); err == nil {
		t.Fatal("closed export accepted a write")
	}
	if err := output.Commit(t.Context()); err == nil {
		t.Fatal("failed sync published the export")
	}
	if err := output.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // Test-owned destination.
	if err != nil || string(data) != "previous\n" {
		t.Fatalf("write failure replaced destination: %q (%v)", data, err)
	}
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".fctl-export-*"))
	if err != nil || len(temps) != 0 {
		t.Fatal("failed export left a temporary file")
	}
}

type finalProgressFailureWriter struct{ writes int }

func (w *finalProgressFailureWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		return len(data), nil
	}
	return 0, io.ErrClosedPipe
}

func TestFinalProgressFailurePreservesExport(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "destination")
	if err := os.WriteFile(path, []byte("previous\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	var failures pluginStreamErrors
	spec := pluginsdk.CommandSpec{Streams: &pluginsdk.StreamSpec{Output: true}, Files: &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}}
	output, err := openPluginStreamOutput(cmd, spec, pluginsdk.ExecuteRequest{Flags: map[string]string{"file": path}}, &failures)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(output.writer, "replacement\n"); err != nil {
		t.Fatal(err)
	}
	progress := &pluginProgress{ctx: t.Context(), writer: &finalProgressFailureWriter{}, enabled: true, failures: &failures, now: time.Now}
	if err := progress.Report(pluginsdk.ProgressEvent{Phase: "export", Completed: 1, Unit: "records"}); err != nil {
		t.Fatal(err)
	}
	if err := completePluginStream(t.Context(), output, progress, nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("final progress failure lost: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // Test-owned destination.
	if err != nil || string(data) != "previous\n" {
		t.Fatalf("final progress failure published the export: %q (%v)", data, err)
	}
}
