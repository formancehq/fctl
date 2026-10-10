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
	"net/url"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

const importBlockSize = 100

// Logs cross the plugin transport as an array or a JSON string containing
// NDJSON. The service still receives historical octet-stream NDJSON batches.
func importLogs(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	ledgerName := r.Args[0]
	if err := validateSegment(ledgerName, "ledger name"); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	logs, err := decodeLogs(r.Body)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	skipped := 0
	if r.Flags["resume-from-last-log"] == "true" {
		skipped, err = resumeOffset(ctx, client, ledgerName, logs)
		if err != nil {
			return response(nil, err)
		}
	}
	progress := importProgress{Skipped: skipped}
	for start := skipped; start < len(logs); start += importBlockSize {
		end := min(start+importBlockSize, len(logs))
		block, err := encodeNDJSON(logs[start:end])
		if err != nil {
			return progress.response(), err
		}
		_, err = client.Do(ctx, http.MethodPost, httpclient.Path("v2", ledgerName, "logs", "import"), nil, block, http.Header{"Content-Type": {"application/octet-stream"}})
		if err != nil {
			if failure, ok := errors.AsType[*httpclient.Error](err); ok {
				progress.Error = failure.Body
			}
			return progress.response(), err
		}
		progress.Imported += end - start
		progress.Batches++
	}
	return progress.response(), nil
}

type importProgress struct {
	Imported int             `json:"imported"`
	Skipped  int             `json:"skipped"`
	Batches  int             `json:"batches"`
	Error    json.RawMessage `json:"error,omitempty"`
}

func (p importProgress) response() pluginsdk.ExecuteResponse {
	data := fmt.Appendf(nil, `{"imported":%d,"skipped":%d,"batches":%d`, p.Imported, p.Skipped, p.Batches)
	if p.Error != nil {
		data = append(data, `,"error":`...)
		data = append(data, p.Error...)
	}
	data = append(data, '}')
	return pluginsdk.ExecuteResponse{Data: data}
}

func encodeNDJSON(logs []json.RawMessage) ([]byte, error) {
	var block []byte
	for _, log := range logs {
		var compact bytes.Buffer
		if err := json.Compact(&compact, log); err != nil {
			return nil, fmt.Errorf("encode log: %w", err)
		}
		block = append(block, compact.Bytes()...)
		block = append(block, '\n')
	}
	return block, nil
}

func resumeOffset(ctx context.Context, client *httpclient.Client, ledgerName string, logs []json.RawMessage) (int, error) {
	data, err := client.Do(ctx, http.MethodGet, httpclient.Path("v2", ledgerName, "logs"), url.Values{"pageSize": {"1"}}, json.RawMessage("null"), nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Cursor struct {
			Data []json.RawMessage `json:"data"`
		} `json:"cursor"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("decode last log: %w", err)
	}
	if len(result.Cursor.Data) == 0 {
		return 0, nil
	}
	id, err := logID(result.Cursor.Data[0])
	if err != nil {
		return 0, err
	}
	for index, log := range logs {
		candidate, err := logID(log)
		if err != nil {
			return 0, err
		}
		if candidate == id {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("log %s not found in import Body", id)
}

func decodeLogs(raw json.RawMessage) ([]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("import Body is required; the host must read file/stdin logs")
	}
	var logs []json.RawMessage
	var ndjson string
	if bytes.TrimSpace(raw)[0] == '"' {
		if err := json.Unmarshal(raw, &ndjson); err != nil {
			return nil, fmt.Errorf("decode NDJSON string: %w", err)
		}
		var err error
		logs, err = decodeNDJSON(strings.NewReader(ndjson))
		if err != nil {
			return nil, err
		}
	} else if err := json.Unmarshal(raw, &logs); err != nil || logs == nil {
		return nil, fmt.Errorf("logs must be a JSON array or a JSON string containing NDJSON")
	}
	for index, log := range logs {
		if _, err := logID(log); err != nil {
			return nil, fmt.Errorf("log %d: %w", index, err)
		}
	}
	return logs, nil
}

func logID(raw json.RawMessage) (string, error) {
	var log map[string]json.RawMessage
	if err := json.Unmarshal(raw, &log); err != nil || log == nil {
		return "", fmt.Errorf("log must be a JSON object")
	}
	id, err := nonnegativeInteger(string(log["id"]), "log ID")
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func decodeNDJSON(reader io.Reader) ([]json.RawMessage, error) {
	buffer := bufio.NewReader(reader)
	logs := make([]json.RawMessage, 0)
	for {
		line, err := buffer.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimSpace(line)
			if len(line) == 0 || !json.Valid(line) {
				return nil, fmt.Errorf("invalid NDJSON log at line %d", len(logs)+1)
			}
			logs = append(logs, json.RawMessage(line))
		}
		if errors.Is(err, io.EOF) {
			return logs, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read logs: %w", err)
		}
	}
}

func exportLogs(ctx context.Context, client *httpclient.Client, ledgerName string) (result pluginsdk.ExecuteResponse, err error) {
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
	data, err := io.ReadAll(io.LimitReader(res.Body, 32<<20+1))
	if err != nil {
		return result, fmt.Errorf("read export: %w", err)
	}
	if len(data) > 32<<20 {
		return result, fmt.Errorf("export exceeds the 32 MiB plugin response limit")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
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
	logs, err := decodeNDJSON(bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	result.Data, err = json.Marshal(logs)
	return result, err
}
