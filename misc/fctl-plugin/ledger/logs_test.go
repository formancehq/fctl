package ledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

//nolint:gocognit // Keep batch request assertions beside the generated 205-log fixture.
func TestImportNDJSONBatchesAndExactNumbers(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if r.Method != "POST" || r.URL.EscapedPath() != "/v2/named/logs/import" || r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("unexpected import request %s %s, content type %q", r.Method, r.URL, r.Header.Get("Content-Type"))
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		lines := bytes.Split(bytes.TrimSuffix(data, []byte{'\n'}), []byte{'\n'})
		wantLines := 100
		if call == 3 {
			wantLines = 5
		}
		if len(lines) != wantLines || len(data) == 0 || data[len(data)-1] != '\n' {
			t.Errorf("batch %d has %d lines, want %d, ending %q", call, len(lines), wantLines, data[len(data)-1:])
		}
		for index, line := range lines {
			want := fmt.Sprintf(`{"id":%d,"payload":{"amount":%s,"unknown":true}}`, int(call-1)*100+index, huge)
			assertJSON(t, line, []byte(want))
		}
		w.WriteHeader(204)
	}))
	t.Cleanup(server.Close)
	var ndjson strings.Builder
	for index := range 205 {
		if _, err := fmt.Fprintf(&ndjson, `{"id":%d,"payload":{"amount":%s,"unknown":true}}`, index, huge); err != nil {
			t.Fatal(err)
		}
		if index < 204 {
			if err := ndjson.WriteByte('\n'); err != nil {
				t.Fatal(err)
			}
		}
	}
	body, err := json.Marshal(ndjson.String())
	if err != nil {
		t.Fatal(err)
	}
	r := executeRequest("import", []string{"named", "host-read.ndjson"}, map[string]string{"file": "host-only"}, string(body))
	r.Endpoint = server.URL
	result, err := New(server.Client()).Execute(t.Context(), r)
	if err != nil || calls.Load() != 3 {
		t.Fatalf("calls %d: %v", calls.Load(), err)
	}
	assertJSON(t, result.Data, []byte(`{"imported":205,"skipped":0,"batches":3}`))
}

//nolint:gocognit // The checkpoint matrix verifies reads, skipped writes and progress in one contract test.
func TestImportArrayResumeAndMissingCheckpoint(t *testing.T) {
	for _, checkpoint := range []string{"", huge, "12", "bad"} {
		t.Run(checkpoint, func(t *testing.T) {
			var lookups, writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					lookups.Add(1)
					if r.URL.Path != "/v2/named/logs" || r.URL.Query().Get("pageSize") != "1" {
						t.Errorf("lookup %s", r.URL)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if string(body) != "null" {
						t.Errorf("historical nil filter body %s", body)
					}
					switch checkpoint {
					case "":
						writeJSON(t, w, `{"cursor":{"data":[]}}`)
					case "bad":
						writeJSON(t, w, `{"cursor":{"data":[{"id":"bad"}]}}`)
					default:
						writeJSON(t, w, `{"cursor":{"data":[{"id":`+checkpoint+`}]}}`)
					}
					return
				}
				writes.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if checkpoint == huge && string(body) != "{\"id\":1234567890123456789012345678901234567891}\n" {
					t.Errorf("resume body %s", body)
				}
				w.WriteHeader(204)
			}))
			t.Cleanup(server.Close)
			r := executeRequest("import", []string{"named"}, map[string]string{"resume-from-last-log": "true"}, `[{"id":`+huge+`},{"id":1234567890123456789012345678901234567891}]`)
			r.Endpoint = server.URL
			result, err := New(server.Client()).Execute(t.Context(), r)
			if lookups.Load() != 1 {
				t.Errorf("lookups %d", lookups.Load())
			}
			if checkpoint == "12" || checkpoint == "bad" {
				if err == nil || writes.Load() != 0 {
					t.Fatalf("expected checkpoint error, writes %d: %v", writes.Load(), err)
				}
				return
			}
			if err != nil || writes.Load() != 1 {
				t.Fatalf("writes %d: %v", writes.Load(), err)
			}
			want := `{"imported":2,"skipped":0,"batches":1}`
			if checkpoint == huge {
				want = `{"imported":1,"skipped":1,"batches":1}`
			}
			assertJSON(t, result.Data, []byte(want))
		})
	}
}

func TestImportPartialResultsNoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 2 {
			w.WriteHeader(409)
			writeJSON(t, w, `{"errorCode":"IMPORT","errorMessage":"bad hash","details":{"id":`+huge+`}}`)
			return
		}
		w.WriteHeader(204)
	}))
	t.Cleanup(server.Close)
	logs := make([]json.RawMessage, 105)
	for index := range logs {
		logs[index] = json.RawMessage(fmt.Sprintf(`{"id":%d}`, index))
	}
	body, err := json.Marshal(logs)
	if err != nil {
		t.Fatal(err)
	}
	r := executeRequest("import", []string{"named"}, nil, string(body))
	r.Endpoint = server.URL
	result, err := New(server.Client()).Execute(t.Context(), r)
	if err == nil || calls.Load() != 2 {
		t.Fatalf("calls %d: %v", calls.Load(), err)
	}
	assertJSON(t, result.Data, []byte(`{"imported":100,"skipped":0,"batches":1,"error":{"errorCode":"IMPORT","errorMessage":"bad hash","details":{"id":`+huge+`}}}`))
}

//nolint:gocognit // Assert the raw export stream and its JSON conversion in the same fixture matrix.
func TestExportNDJSONAndHostOwnsFile(t *testing.T) {
	for _, content := range []string{"", `{"id":` + huge + `,"payload":{"amount":` + huge + `}}` + "\n" + `{"id":1234567890123456789012345678901234567891}`} {
		t.Run(fmt.Sprint(len(content)), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.EscapedPath() != "/gateway/v2/named/logs/export" || r.Header.Get("Accept") != "*/*" {
					t.Errorf("export request %s %s accept %s", r.Method, r.URL, r.Header.Get("Accept"))
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				if _, err := io.WriteString(w, content); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			r := executeRequest("export", nil, map[string]string{"ledger": "named", "file": "/this/path/must/not/be/written"}, "")
			r.Endpoint = server.URL + "/gateway"
			result, err := New(server.Client()).Execute(t.Context(), r)
			if err != nil {
				t.Fatal(err)
			}
			want := "[]"
			if content != "" {
				want = "[" + strings.ReplaceAll(content, "\n", ",") + "]"
			}
			assertJSON(t, result.Data, []byte(want))
		})
	}
}

func TestExportRejectsMalformedStreamAndPreservesError(t *testing.T) {
	for _, status := range []int{200, 403} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			body := "malformed\n"
			if status == 403 {
				body = `{"errorCode":"DENIED","errorMessage":"no export"}`
			}
			if _, err := io.WriteString(w, body); err != nil {
				t.Error(err)
			}
		}))
		r := executeRequest("export", nil, nil, "")
		r.Endpoint = server.URL
		result, err := New(server.Client()).Execute(t.Context(), r)
		if err == nil {
			t.Error("expected export error")
		}
		if status == 403 && string(result.Data) != `{"errorCode":"DENIED","errorMessage":"no export"}` {
			t.Errorf("error data %s", result.Data)
		}
		server.Close()
	}
}

func TestReadLogsUnboundedLinesAndEmptyInput(t *testing.T) {
	large := strings.Repeat("a", 128<<10)
	body, err := json.Marshal(`{"id":1,"large":"` + large + `"}`)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := decodeLogs(body)
	if err != nil || len(logs) != 1 || !bytes.Contains(logs[0], []byte(large)) {
		t.Fatalf("logs %d: %v", len(logs), err)
	}
	if _, err := decodeNDJSON(brokenReader{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error %v", err)
	}
	for _, body := range []string{`[]`, `""`} {
		logs, err := decodeLogs([]byte(body))
		if err != nil || len(logs) != 0 {
			t.Fatalf("empty logs %s: %v", body, err)
		}
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
