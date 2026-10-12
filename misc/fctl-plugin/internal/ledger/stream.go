package ledger

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

const (
	maxLogRecordBytes   = 4 << 20
	maxImportBatchBytes = 4 << 20 // The callback includes the NDJSON delimiter.
)

var errEndLogs = errors.New("end of NDJSON logs")

// Import holds at most one bounded batch and one bounded record. Read progress
// measures source bytes; import progress measures only acknowledged log writes.
func importLogsStream(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
	progress := importProgress{}
	ledgerName := r.Args[0]
	if err := validateSegment(ledgerName, "ledger name"); err != nil {
		return progress.response(), err
	}
	if streams.Input == nil || streams.Output != nil {
		return progress.response(), fmt.Errorf("import requires an input stream and no output stream")
	}
	if err := reportProgress(ctx, streams, "read", 0, streams.InputSize, "bytes"); err != nil {
		return progress.response(), err
	}
	if err := reportProgress(ctx, streams, "import", 0, nil, "logs"); err != nil {
		return progress.response(), err
	}
	checkpoint := ""
	if r.Flags["resume-from-last-log"] == "true" {
		var err error
		checkpoint, err = latestLogID(ctx, client, ledgerName)
		if err != nil {
			return progress.response(), err
		}
	}
	err := importStreamRecords(ctx, client, ledgerName, streams, checkpoint, &progress)
	return progress.response(), err
}

func importStreamRecords(ctx context.Context, client *httpclient.Client, ledgerName string, streams pluginsdk.ExecutionStreams, checkpoint string, progress *importProgress) error {
	reader := bufio.NewReader(&progressReader{ctx: ctx, streams: streams})
	batch := logBatch{client: client, ledger: ledgerName, streams: streams}
	var scanned int64
	for {
		record, err := readLogRecord(ctx, reader)
		if errors.Is(err, errEndLogs) {
			break
		}
		if err != nil {
			return err
		}
		id, err := logID(record)
		if err != nil {
			return fmt.Errorf("log %d: %w", scanned+1, err)
		}
		scanned++
		if checkpoint != "" {
			if id == checkpoint {
				checkpoint = ""
				progress.Skipped = scanned
			}
			continue
		}
		if err := batch.append(ctx, record, progress); err != nil {
			return err
		}
	}
	if checkpoint != "" {
		return fmt.Errorf("log %s not found in import stream; no logs written", checkpoint)
	}
	return batch.flush(ctx, progress)
}

type logBatch struct {
	client  *httpclient.Client
	ledger  string
	streams pluginsdk.ExecutionStreams
	data    []byte
	count   int64
}

func (b *logBatch) append(ctx context.Context, record []byte, progress *importProgress) error {
	var compact bytes.Buffer
	if err := json.Compact(&compact, record); err != nil {
		return fmt.Errorf("encode log: %w", err)
	}
	if compact.Len()+1 > maxImportBatchBytes {
		return fmt.Errorf("NDJSON log with newline exceeds the 4 MiB import batch limit")
	}
	if len(b.data)+compact.Len()+1 > maxImportBatchBytes {
		if err := b.flush(ctx, progress); err != nil {
			return err
		}
	}
	b.data = append(b.data, compact.Bytes()...)
	b.data = append(b.data, '\n')
	b.count++
	if b.count == importBlockSize {
		return b.flush(ctx, progress)
	}
	return nil
}

func (b *logBatch) flush(ctx context.Context, progress *importProgress) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.count == 0 {
		return nil
	}
	_, err := b.client.Do(ctx, http.MethodPost, httpclient.Path("v2", b.ledger, "logs", "import"), nil, b.data, http.Header{"Content-Type": {"application/octet-stream"}})
	if err != nil {
		if failure, ok := errors.AsType[*httpclient.Error](err); ok {
			progress.Error = failure.Body
		}
		// An unacknowledged batch can have reached the service. Never replay it.
		return err
	}
	progress.Imported += b.count
	progress.Batches++
	b.data = b.data[:0]
	b.count = 0
	return reportProgress(ctx, b.streams, "import", progress.Imported, nil, "logs")
}

type progressReader struct {
	ctx       context.Context
	streams   pluginsdk.ExecutionStreams
	completed int64
}

func (r *progressReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.streams.Input.Read(p)
	r.completed += int64(n)
	if n > 0 {
		if reportErr := reportProgress(r.ctx, r.streams, "read", r.completed, r.streams.InputSize, "bytes"); reportErr != nil {
			return 0, fmt.Errorf("report read progress: %w", errors.Join(err, reportErr))
		}
	}
	return n, err
}

func reportProgress(ctx context.Context, streams pluginsdk.ExecutionStreams, phase string, completed int64, total *int64, unit string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	event := pluginsdk.ProgressEvent{Phase: phase, Completed: completed, Total: total, Unit: unit}
	if err := event.Validate(); err != nil {
		return err
	}
	if streams.Report != nil {
		return errors.Join(streams.Report(event), ctx.Err())
	}
	return nil
}

// ReadSlice bounds allocation even for a peer that never sends a newline. The
// returned record keeps the exact bytes, including whitespace and its delimiter.
func readLogRecord(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
	var record []byte
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fragment, err := reader.ReadSlice('\n')
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if len(record)+len(fragment) > maxLogRecordBytes {
			return nil, fmt.Errorf("NDJSON log exceeds the 4 MiB record limit")
		}
		record = append(record, fragment...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		// Only the Reader's exact EOF sentinel ends a record. In particular,
		// a reporter error wrapping EOF must still fail the invocation.
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read logs: %w", err)
		}
		if len(record) == 0 && err == io.EOF {
			return nil, errEndLogs
		}
		return record, nil
	}
}

func exportLogsStream(ctx context.Context, client *httpclient.Client, ledgerName string, streams pluginsdk.ExecutionStreams) (result pluginsdk.ExecuteResponse, err error) {
	if streams.Output == nil || streams.Input != nil || streams.InputSize != nil {
		return result, fmt.Errorf("export requires an output stream and no input stream")
	}
	if err := reportProgress(ctx, streams, "export", 0, nil, "logs"); err != nil {
		return result, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.Endpoint()+httpclient.Path("v2", ledgerName, "logs", "export"), nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Accept", "*/*")
	res, err := client.HTTPClient().Do(req)
	if err != nil {
		return result, fmt.Errorf("export logs: %w", err)
	}
	defer func() { err = errors.Join(err, res.Body.Close()) }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return exportStreamFailure(res)
	}
	var progress exportProgress
	err = copyExportStream(ctx, res.Body, streams, &progress)
	return progress.response(), err
}

type exportProgress struct{ exported, written int64 }

func (p exportProgress) response() pluginsdk.ExecuteResponse {
	return pluginsdk.ExecuteResponse{Data: fmt.Appendf(nil, `{"exported":%d,"bytes":%d}`, p.exported, p.written)}
}

func copyExportStream(ctx context.Context, input io.Reader, streams pluginsdk.ExecutionStreams, progress *exportProgress) error {
	reader := bufio.NewReader(input)
	for {
		record, readErr := readLogRecord(ctx, reader)
		if errors.Is(readErr, errEndLogs) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
		if _, err := logID(record); err != nil {
			return fmt.Errorf("export log %d: %w", progress.exported+1, err)
		}
		if err := progress.writeRecord(ctx, streams.Output, record); err != nil {
			return err
		}
		if err := reportProgress(ctx, streams, "export", progress.exported, nil, "logs"); err != nil {
			return err
		}
	}
}

func (p *exportProgress) writeRecord(ctx context.Context, output io.Writer, record []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n, err := output.Write(record)
	p.written += int64(n)
	if err != nil {
		return fmt.Errorf("write export: %w", err)
	}
	if n != len(record) {
		return io.ErrShortWrite
	}
	p.exported++
	return nil
}

func exportStreamFailure(res *http.Response) (pluginsdk.ExecuteResponse, error) {
	data, err := io.ReadAll(io.LimitReader(res.Body, maxLogRecordBytes+1))
	if err != nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("read export error: %w", err)
	}
	if len(data) > maxLogRecordBytes {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("export error response exceeds 4 MiB")
	}
	failure := &httpclient.Error{StatusCode: res.StatusCode}
	var details struct {
		Code    string `json:"errorCode"`
		Message string `json:"errorMessage"`
	}
	if json.Unmarshal(data, &details) == nil {
		failure.Body, failure.Code, failure.Message = data, details.Code, details.Message
	}
	return response(nil, failure)
}
