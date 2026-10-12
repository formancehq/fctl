package ledger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type streamTransport func(*http.Request) (*http.Response, error)

func (f streamTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func streamResponse(status int, body io.Reader) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(body)}
}

func streamPlugin(t *testing.T, transport streamTransport) pluginsdk.StreamingPlugin {
	t.Helper()
	streaming, ok := New(&http.Client{Transport: transport}).(pluginsdk.StreamingPlugin)
	if !ok {
		t.Fatal("Ledger must implement StreamingPlugin")
	}
	return streaming
}

func streamRequest(command string, args []string, flags map[string]string, body string) pluginsdk.ExecuteRequest {
	r := executeRequest(command, args, flags, body)
	r.Endpoint = "https://ledger.example"
	return r
}

// The fixture never allocates the whole input. It repeats one bounded record.
type generatedLogs struct {
	record    []byte
	remaining int
	offset    int
	eof       bool
	endError  error
}

func (g *generatedLogs) Read(p []byte) (int, error) {
	if g.remaining == 0 {
		g.eof = true
		if g.endError != nil {
			return 0, g.endError
		}
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && g.remaining > 0 {
		copied := copy(p[n:], g.record[g.offset:])
		n += copied
		g.offset += copied
		if g.offset == len(g.record) {
			g.offset = 0
			g.remaining--
		}
	}
	return n, nil
}

//nolint:gocognit // Verify byte and count bounds, raw fidelity and both progress units beside the generated source.
func TestStreamImportOver64MiBBoundedBatches(t *testing.T) {
	const count = 550
	record := []byte(`{"id":` + huge + `,"unknown":{"amount":` + huge + `,"text":"` + strings.Repeat("x", 128<<10) + `"}}` + "\n")
	input := &generatedLogs{record: record, remaining: count}
	size := int64(len(record) * count)
	if size <= 64<<20 {
		t.Fatal("fixture must exceed the old total stream limit")
	}
	var posts, acknowledged int64
	var lastRead, lastImported int64
	p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/v2/named/logs/import" || r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		if posts == 0 && input.eof {
			t.Fatal("first import waited for source EOF")
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, maxImportBatchBytes+1))
		if err != nil || len(data) > maxImportBatchBytes {
			t.Fatalf("batch bytes %d: %v", len(data), err)
		}
		logs := bytes.Count(data, []byte{'\n'})
		if logs == 0 || logs > importBlockSize || !bytes.Equal(data, bytes.Repeat(record, logs)) {
			t.Fatal("batch lost raw fields, exact numbers or its bound")
		}
		acknowledged += int64(logs)
		posts++
		return streamResponse(204, strings.NewReader("")), nil
	})
	r := streamRequest("import", []string{"named"}, nil, "")
	result, err := p.ExecuteStream(t.Context(), r, pluginsdk.ExecutionStreams{
		Input: input, InputSize: &size, Report: func(event pluginsdk.ProgressEvent) error {
			if err := event.Validate(); err != nil {
				return err
			}
			switch event.Phase {
			case "read":
				if event.Unit != "bytes" || event.Total == nil || *event.Total != size || event.Completed < lastRead {
					t.Fatalf("invalid read progress %+v", event)
				}
				lastRead = event.Completed
			case "import":
				if event.Unit != "logs" || event.Total != nil || event.Completed != acknowledged {
					t.Fatalf("import progress must count ACKs: %+v, ACKs %d", event, acknowledged)
				}
				lastImported = event.Completed
			default:
				t.Fatalf("unexpected phase %+v", event)
			}
			return nil
		},
	})
	if err != nil || lastRead != size || lastImported != count || posts <= count/importBlockSize {
		t.Fatalf("read %d imported %d posts %d: %v", lastRead, lastImported, posts, err)
	}
	assertJSON(t, result.Data, fmt.Appendf(nil, `{"imported":%d,"skipped":0,"batches":%d}`, count, posts))
}

//nolint:gocognit // Keep the checkpoint matrix and its no-write assertions together.
func TestStreamImportResumeExactCheckpoint(t *testing.T) {
	for _, checkpoint := range []string{"", huge, "42", `"invalid"`} {
		t.Run(checkpoint, func(t *testing.T) {
			var posts int
			p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == "GET" {
					if r.URL.Path != "/v2/named/logs" || r.URL.Query().Get("pageSize") != "1" {
						t.Fatalf("checkpoint request %s", r.URL)
					}
					body := `{"cursor":{"data":[]}}`
					if checkpoint != "" {
						body = `{"cursor":{"data":[{"id":` + checkpoint + `}]}}`
					}
					return streamResponse(200, strings.NewReader(body)), nil
				}
				posts++
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if checkpoint == huge && string(data) != "{\"id\":0}\n" {
					t.Fatalf("replayed checkpoint %s", data)
				}
				return streamResponse(204, strings.NewReader("")), nil
			})
			r := streamRequest("import", []string{"named"}, map[string]string{"resume-from-last-log": "true"}, "")
			input := strings.NewReader(`{"id":1}` + "\n" + `{"id":` + huge + `}` + "\n" + `{"id":0}`)
			result, err := p.ExecuteStream(t.Context(), r, pluginsdk.ExecutionStreams{Input: input})
			if checkpoint == "42" || checkpoint == `"invalid"` {
				if err == nil || posts != 0 {
					t.Fatalf("missing checkpoint made %d writes: %v", posts, err)
				}
				assertJSON(t, result.Data, []byte(`{"imported":0,"skipped":0,"batches":0}`))
				return
			}
			if err != nil || posts != 1 {
				t.Fatalf("posts %d: %v", posts, err)
			}
			want := `{"imported":3,"skipped":0,"batches":1}`
			if checkpoint == huge {
				want = `{"imported":1,"skipped":2,"batches":1}`
			}
			assertJSON(t, result.Data, []byte(want))
		})
	}
}

//nolint:gocognit // Verify both structured and ambiguous failures against the same ACK counters.
func TestStreamImportFailedBatchNeverReplayed(t *testing.T) {
	for _, transportError := range []bool{false, true} {
		t.Run(fmt.Sprint(transportError), func(t *testing.T) {
			failure := errors.New("lost acknowledgment")
			posts := 0
			p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
				posts++
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					t.Fatal(err)
				}
				if posts == 2 {
					if transportError {
						return nil, failure
					}
					return streamResponse(409, strings.NewReader(`{"errorCode":"IMPORT","errorMessage":"bad hash"}`)), nil
				}
				return streamResponse(204, strings.NewReader("")), nil
			})
			input := &generatedLogs{record: []byte("{\"id\":1}\n"), remaining: 205}
			result, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: input})
			if err == nil || posts != 2 {
				t.Fatalf("posts %d: %v", posts, err)
			}
			if transportError && !errors.Is(err, failure) {
				t.Fatal(err)
			}
			want := `{"imported":100,"skipped":0,"batches":1}`
			if !transportError {
				want = `{"imported":100,"skipped":0,"batches":1,"error":{"errorCode":"IMPORT","errorMessage":"bad hash"}}`
			}
			assertJSON(t, result.Data, []byte(want))
		})
	}
}

func TestStreamImportLateInputFailureCountsOnlyACKs(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprint(malformed), func(t *testing.T) {
			posts := 0
			p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
				posts++
				return streamResponse(204, strings.NewReader("")), nil
			})
			var input io.Reader = &generatedLogs{record: []byte("{\"id\":1}\n"), remaining: 100, endError: io.ErrUnexpectedEOF}
			if malformed {
				input = io.MultiReader(&generatedLogs{record: []byte("{\"id\":1}\n"), remaining: 100}, strings.NewReader("{bad}\n"))
			}
			result, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: input})
			if err == nil || posts != 1 {
				t.Fatalf("posts %d: %v", posts, err)
			}
			if !malformed && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatal(err)
			}
			assertJSON(t, result.Data, []byte(`{"imported":100,"skipped":0,"batches":1}`))
		})
	}
}

type finalErrorReader struct {
	content string
	err     error
}

func (r *finalErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.content)
	r.content = r.content[n:]
	return n, r.err
}

//nolint:gocognit // Keep reporter and cancellation outcomes beside their confirmed-write assertions.
func TestStreamImportProgressErrorsAndCancellation(t *testing.T) {
	for _, phase := range []string{"read", "import"} {
		for _, cancelContext := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", phase, cancelContext), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				failure := errors.New("report stopped")
				posts := 0
				p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
					posts++
					return streamResponse(204, strings.NewReader("")), nil
				})
				input := &finalErrorReader{content: "{\"id\":1}", err: io.EOF}
				result, err := p.ExecuteStream(ctx, streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: input, Report: func(event pluginsdk.ProgressEvent) error {
					if event.Phase == phase && event.Completed > 0 {
						if cancelContext {
							cancel()
							return nil
						}
						return failure
					}
					return nil
				}})
				wantErr := failure
				if cancelContext {
					wantErr = context.Canceled
				}
				if !errors.Is(err, wantErr) {
					t.Fatalf("got %v want %v", err, wantErr)
				}
				want := `{"imported":0,"skipped":0,"batches":0}`
				if phase == "import" {
					want = `{"imported":1,"skipped":0,"batches":1}`
					if posts != 1 {
						t.Fatalf("posts %d", posts)
					}
				} else if posts != 0 {
					t.Fatalf("read failure made %d writes", posts)
				}
				assertJSON(t, result.Data, []byte(want))
			})
		}
	}
}

func TestStreamImportEmptyAndZeroTotal(t *testing.T) {
	for _, total := range []*int64{nil, new(int64(0)), new(int64(-1))} {
		p := streamPlugin(t, func(*http.Request) (*http.Response, error) { t.Fatal("empty import must not write"); return nil, nil })
		var events []pluginsdk.ProgressEvent
		result, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: strings.NewReader(""), InputSize: total, Report: func(e pluginsdk.ProgressEvent) error { events = append(events, e); return nil }})
		if total != nil && *total < 0 {
			if err == nil {
				t.Fatal("negative size accepted")
			}
			continue
		}
		if err != nil || len(events) != 2 || events[0].Total != total || events[0].Completed != 0 {
			t.Fatalf("events %+v: %v", events, err)
		}
		assertJSON(t, result.Data, []byte(`{"imported":0,"skipped":0,"batches":0}`))
	}
}

func TestStreamImportReportEOFIsFailure(t *testing.T) {
	p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("report failure must not write")
		return nil, nil
	})
	input := &finalErrorReader{content: "{\"id\":1}", err: io.EOF}
	result, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{
		Input: input, Report: func(e pluginsdk.ProgressEvent) error {
			if e.Phase == "read" && e.Completed > 0 {
				return io.EOF
			}
			return nil
		},
	})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("report EOF swallowed: %v", err)
	}
	assertJSON(t, result.Data, []byte(`{"imported":0,"skipped":0,"batches":0}`))
}

func TestStreamImportZeroTotalRejectsNonemptyInput(t *testing.T) {
	p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid total must not write")
		return nil, nil
	})
	_, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{
		Input: strings.NewReader("{\"id\":1}"), InputSize: new(int64(0)),
	})
	if err == nil {
		t.Fatal("nonempty source accepted with zero total")
	}
}

func TestStreamImportCancelDuringRequestDoesNotACKOrRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	posts := 0
	p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
		posts++
		cancel()
		return nil, r.Context().Err()
	})
	result, err := p.ExecuteStream(ctx, streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: strings.NewReader("{\"id\":1}")})
	if !errors.Is(err, context.Canceled) || posts != 1 {
		t.Fatalf("posts %d: %v", posts, err)
	}
	assertJSON(t, result.Data, []byte(`{"imported":0,"skipped":0,"batches":0}`))
}

func TestStreamImportInvalidCheckpointEnvelopeNeverWrites(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"cursor":{}}`, `{"cursor":{"data":null}}`} {
		t.Run(body, func(t *testing.T) {
			p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("invalid checkpoint response must not write")
				}
				return streamResponse(200, strings.NewReader(body)), nil
			})
			_, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, map[string]string{"resume-from-last-log": "true"}, ""), pluginsdk.ExecutionStreams{Input: strings.NewReader("{\"id\":1}")})
			if err == nil {
				t.Fatal("invalid checkpoint response treated as empty ledger")
			}
		})
	}
}

func TestStreamRecordAndBatchByteLimits(t *testing.T) {
	for _, length := range []int{maxLogRecordBytes, maxLogRecordBytes + 1} {
		input := `{"id":1,"payload":"` + strings.Repeat("x", length-len(`{"id":1,"payload":""}`)) + `"}`
		p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
			t.Fatal("oversize import must not write")
			return nil, nil
		})
		_, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: strings.NewReader(input)})
		if err == nil || !strings.Contains(err.Error(), "4 MiB") {
			t.Fatal(err)
		}
	}
	// Exactly 4 MiB including the added delimiter is accepted.
	input := `{"id":1,"payload":"` + strings.Repeat("x", maxLogRecordBytes-1-len(`{"id":1,"payload":""}`)) + `"}`
	posts := 0
	p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
		posts++
		data, err := io.ReadAll(r.Body)
		if err != nil || len(data) != maxImportBatchBytes {
			t.Fatalf("bytes %d: %v", len(data), err)
		}
		return streamResponse(204, strings.NewReader("")), nil
	})
	_, err := p.ExecuteStream(t.Context(), streamRequest("import", []string{"named"}, nil, ""), pluginsdk.ExecutionStreams{Input: strings.NewReader(input)})
	if err != nil || posts != 1 {
		t.Fatalf("posts %d: %v", posts, err)
	}
}

type earlyWriter struct {
	writer io.Writer
	source *generatedLogs
	early  bool
	bytes  int64
}

func (w *earlyWriter) Write(p []byte) (int, error) {
	if !w.source.eof {
		w.early = true
	}
	n, err := w.writer.Write(p)
	w.bytes += int64(n)
	return n, err
}

func TestStreamExportOver64MiBToFileBeforeEOF(t *testing.T) {
	const count = 550
	record := []byte(`{"id":` + huge + `,"unknown":"` + strings.Repeat("x", 128<<10) + `"}` + "\n")
	source := &generatedLogs{record: record, remaining: count}
	p := streamPlugin(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/v2/named/logs/export" || r.Header.Get("Accept") != "*/*" {
			t.Fatalf("request %s", r.URL)
		}
		return streamResponse(200, source), nil
	})
	file, err := os.Create(filepath.Join(t.TempDir(), "export.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	output := &earlyWriter{writer: file, source: source}
	result, err := p.ExecuteStream(t.Context(), streamRequest("export", nil, map[string]string{"ledger": "named"}, ""), pluginsdk.ExecutionStreams{Output: output})
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size := int64(len(record) * count)
	if size <= 64<<20 || info.Size() != size || !output.early || !source.eof {
		t.Fatalf("size %d early %t eof %t", info.Size(), output.early, source.eof)
	}
	assertJSON(t, result.Data, fmt.Appendf(nil, `{"exported":%d,"bytes":%d}`, count, size))
}

func TestStreamExportPreservesExactRawBytesAndMissingNewline(t *testing.T) {
	for _, content := range []string{"", " {\"id\":" + huge + ",\"unknown\":{\"amount\":" + huge + "}}\r\n{\"id\":0}"} {
		p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
			return streamResponse(200, strings.NewReader(content)), nil
		})
		var output bytes.Buffer
		result, err := p.ExecuteStream(t.Context(), streamRequest("export", nil, nil, ""), pluginsdk.ExecutionStreams{Output: &output})
		if err != nil || output.String() != content {
			t.Fatalf("raw bytes %q: %v", output.String(), err)
		}
		count := 0
		if content != "" {
			count = 2
		}
		assertJSON(t, result.Data, fmt.Appendf(nil, `{"exported":%d,"bytes":%d}`, count, len(content)))
	}
}

type errorWriter struct {
	err   error
	short bool
}

func (w errorWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, w.err
}

//nolint:gocognit // Keep each injected IO failure beside its output prefix and summary assertions.
func TestStreamExportFailuresAndCancellation(t *testing.T) {
	failure := errors.New("output failed")
	for _, kind := range []string{"malformed", "read", "write", "short", "report", "cancel", "http", "oversize", "close"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var output bytes.Buffer
			streams := pluginsdk.ExecutionStreams{Output: &output}
			var body io.Reader = strings.NewReader("{\"id\":1}\n{\"id\":2}")
			status := 200
			switch kind {
			case "malformed":
				body = strings.NewReader("{\"id\":1}\n{bad}\n")
			case "read":
				body = io.MultiReader(strings.NewReader("{\"id\":1}\n"), brokenReader{})
			case "write":
				streams.Output = errorWriter{err: failure}
			case "short":
				streams.Output = errorWriter{short: true}
			case "report", "cancel":
				streams.Report = func(e pluginsdk.ProgressEvent) error {
					if e.Completed > 0 {
						if kind == "cancel" {
							cancel()
							return nil
						}
						return failure
					}
					return nil
				}
			case "http":
				status = 403
				body = strings.NewReader(`{"errorCode":"DENIED","errorMessage":"no export"}`)
			case "oversize":
				body = strings.NewReader(strings.Repeat("x", maxLogRecordBytes+1))
			}
			p := streamPlugin(t, func(*http.Request) (*http.Response, error) {
				res := streamResponse(status, body)
				if kind == "close" {
					res.Body = closeErrorBody{Reader: body, err: failure}
				}
				return res, nil
			})
			result, err := p.ExecuteStream(ctx, streamRequest("export", nil, nil, ""), streams)
			if err == nil {
				t.Fatal("expected failure")
			}
			switch kind {
			case "write", "report", "close":
				if !errors.Is(err, failure) {
					t.Fatal(err)
				}
			case "short":
				if !errors.Is(err, io.ErrShortWrite) {
					t.Fatal(err)
				}
			case "cancel":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			case "read":
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatal(err)
				}
			case "http":
				assertJSON(t, result.Data, []byte(`{"errorCode":"DENIED","errorMessage":"no export"}`))
				if output.Len() != 0 {
					t.Fatal("HTTP error leaked into NDJSON")
				}
			}
			if kind == "malformed" || kind == "read" || kind == "report" || kind == "cancel" {
				if output.String() != "{\"id\":1}\n" {
					t.Fatalf("invalid record written %q", output.String())
				}
				assertJSON(t, result.Data, []byte(`{"exported":1,"bytes":9}`))
			}
		})
	}
}

type closeErrorBody struct {
	io.Reader
	err error
}

func (b closeErrorBody) Close() error { return b.err }
