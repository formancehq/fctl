package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func (h *helperPlugin) ExecuteStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (result pluginsdk.ExecuteResponse, err error) {
	if request.Flags["action"] == "reject" {
		return pluginsdk.ExecuteResponse{Data: []byte(`{"imported":0}`)}, fmt.Errorf("rejected before input")
	}
	if request.Flags["action"] == "invalid-progress" {
		return pluginsdk.ExecuteResponse{}, streams.Report(pluginsdk.ProgressEvent{Phase: "\x1b[2J", Unit: "bytes"})
	}
	if request.Flags["action"] == "cancel" {
		<-ctx.Done()
		return pluginsdk.ExecuteResponse{}, ctx.Err()
	}
	if request.Flags["action"] == "http-stream" {
		return h.executeHTTPStream(ctx, request, streams)
	}
	return copyHelperStream(streams)
}

func (h *helperPlugin) executeHTTPStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (result pluginsdk.ExecuteResponse, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, request.Endpoint+"/export", streams.Input)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	response, err := h.http.Do(req)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	n, err := io.Copy(streams.Output, response.Body)
	return pluginsdk.ExecuteResponse{Data: fmt.Appendf(nil, `{"bytes":%d}`, n)}, err
}

func copyHelperStream(streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
	var n int64
	buffer := make([]byte, 32<<10)
	for {
		read, err := streams.Input.Read(buffer)
		if read > 0 {
			written, writeErr := streams.Output.Write(buffer[:read])
			n += int64(written)
			if writeErr != nil {
				return pluginsdk.ExecuteResponse{}, writeErr
			}
			if err := streams.Report(pluginsdk.ProgressEvent{Phase: "copy", Unit: "bytes", Completed: n, Total: streams.InputSize}); err != nil {
				return pluginsdk.ExecuteResponse{}, err
			}
		}
		if errors.Is(err, io.EOF) {
			return pluginsdk.ExecuteResponse{Data: fmt.Appendf(nil, `{"bytes":%d}`, n)}, nil
		}
		if err != nil {
			return pluginsdk.ExecuteResponse{Data: fmt.Appendf(nil, `{"bytes":%d}`, n)}, err
		}
	}
}

// This reader generates large inputs without allocating the complete payload.
type countedReader struct {
	remaining int64
	read      int64
}

func (r *countedReader) Read(target []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := int(min(int64(len(target)), r.remaining))
	for i := range n {
		target[i] = "abcdefghijklmnopqrstuvwxyz"[(r.read+int64(i))%26]
	}
	r.remaining -= int64(n)
	r.read += int64(n)
	return n, nil
}

type checkedWriter struct{ written int64 }

func (w *checkedWriter) Write(data []byte) (int, error) {
	for i, value := range data {
		if value != "abcdefghijklmnopqrstuvwxyz"[(w.written+int64(i))%26] {
			return 0, fmt.Errorf("stream content changed at %d", w.written+int64(i))
		}
	}
	w.written += int64(len(data))
	return len(data), nil
}

func TestExecutableStreamsLargeDataAndProgress(t *testing.T) {
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	const size = int64(80 << 20)
	reader := &countedReader{remaining: size}
	writer := &checkedWriter{}
	var last int64
	updates := 0
	result, err := client.ExecuteStream(t.Context(), helperRequest("copy"), pluginsdk.ExecutionStreams{
		Input: reader, InputSize: new(size), Output: writer,
		Report: func(event pluginsdk.ProgressEvent) error {
			if event.Completed <= last || event.Total == nil || *event.Total != size {
				return fmt.Errorf("invalid progress counter %+v", event)
			}
			if reader.read-writer.written > streamChunkBytes {
				return fmt.Errorf("input was buffered before output")
			}
			last = event.Completed
			updates++
			return nil
		},
	})
	if err != nil || writer.written != size || last != size || updates < 2 || string(result.Data) != `{"bytes":83886080}` {
		t.Fatalf("stream bytes=%d progress=%d updates=%d result=%s err=%v", writer.written, last, updates, result.Data, err)
	}
}

func TestStreamingDoesNotReadBeforeBusinessChecks(t *testing.T) {
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	reader := &countedReader{remaining: 10 << 20}
	result, err := client.ExecuteStream(t.Context(), helperRequest("reject"), pluginsdk.ExecutionStreams{Input: reader})
	if err == nil || reader.read != 0 || string(result.Data) != `{"imported":0}` {
		t.Fatalf("rejected execution read=%d result=%s err=%v", reader.read, result.Data, err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestStreamWriterAndReporterErrorsStopExecution(t *testing.T) {
	for _, failure := range []string{"writer", "reporter", "invalid-progress"} {
		t.Run(failure, func(t *testing.T) {
			client := openHelper(t, http.DefaultClient, "https://unused.example")
			streams := pluginsdk.ExecutionStreams{Input: strings.NewReader("payload"), Output: io.Discard}
			action := "copy"
			switch failure {
			case "writer":
				streams.Output = failingWriter{}
			case "reporter":
				streams.Report = func(pluginsdk.ProgressEvent) error { return io.ErrClosedPipe }
			default:
				action = failure
			}
			if _, err := client.ExecuteStream(t.Context(), helperRequest(action), streams); err == nil {
				t.Fatal("stream failure lost")
			}
		})
	}
}

func TestStreamCancellationReapsExecutable(t *testing.T) {
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err := client.ExecuteStream(ctx, helperRequest("cancel"), pluginsdk.ExecutionStreams{})
	if !errors.Is(err, context.DeadlineExceeded) || !client.process.Exited() {
		t.Fatalf("cancellation err=%v exited=%v", err, client.process.Exited())
	}
}

type failedStreamReader struct{ sent bool }

func (r *failedStreamReader) Read(data []byte) (int, error) {
	if r.sent {
		return 0, io.ErrUnexpectedEOF
	}
	r.sent = true
	return copy(data, "partial"), io.ErrUnexpectedEOF
}

func TestStreamReadFailurePreservesPartialResult(t *testing.T) {
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	result, err := client.ExecuteStream(t.Context(), helperRequest("copy"), pluginsdk.ExecutionStreams{
		Input: &failedStreamReader{}, Output: io.Discard,
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) || string(result.Data) != `{"bytes":7}` {
		t.Fatalf("source failure result=%s err=%v", result.Data, err)
	}
}

func TestUnaryOnlyPluginRejectsStreamingWithoutReading(t *testing.T) {
	manifest := helperManifest()
	client := inProcessPlugin(t, func(*http.Client) pluginsdk.Plugin { return &metadataPlugin{manifest: manifest} }, http.DefaultClient, "https://unused.example")
	input := &countedReader{remaining: 1 << 20}
	_, err := client.ExecuteStream(t.Context(), helperRequest("copy"), pluginsdk.ExecutionStreams{Input: input})
	if status.Code(err) != codes.Unimplemented || input.read != 0 {
		t.Fatalf("unsupported stream read=%d err=%v", input.read, err)
	}
}
