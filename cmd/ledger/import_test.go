package ledger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func TestImportLogs(t *testing.T) {
	t.Parallel()

	// Larger than bufio.Scanner's 64 KB default token size.
	longLog := fmt.Sprintf(`{"id":1,"data":"%s"}`, strings.Repeat("a", 200*1024))
	input := `{"id":0}` + "\n" + longLog + "\n" + `{"id":2}`

	var (
		blocks []string
		read   int
	)
	err := importLogs(strings.NewReader(input), 2, func(n int) {
		read += n
	}, func(buffer *bytes.Buffer) error {
		blocks = append(blocks, buffer.String())
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, len(input), read)
	require.Equal(t, []string{
		`{"id":0}` + "\n" + longLog + "\n",
		`{"id":2}` + "\n",
	}, blocks)
}

func TestImportLogsSendError(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("boom")
	err := importLogs(strings.NewReader(`{"id":0}`+"\n"), 100, func(int) {}, func(*bytes.Buffer) error {
		return sendErr
	})
	require.ErrorIs(t, err, sendErr)
}

func TestOpenFileWithOffset(t *testing.T) {
	t.Parallel()

	longLog := fmt.Sprintf(`{"id":1,"data":"%s"}`, strings.Repeat("a", 200*1024))
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"id":0}`+"\n"+longLog+"\n"+`{"id":2}`+"\n"), 0o600))

	c := NewImportController()

	f, offset, err := c.openFileWithOffset(path, big.NewInt(1))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	require.Equal(t, len(`{"id":0}`)+1+len(longLog)+1, offset)

	_, _, err = c.openFileWithOffset(path, big.NewInt(42))
	require.ErrorContains(t, err, "log 42 not found")
}

func TestLastImportLogIDSchemaQueryParams(t *testing.T) {
	t.Parallel()
	const body = `{"cursor":{"pageSize":1,"hasMore":false,"data":[{"id":9007199254740993,"type":"INSERTED_SCHEMA","hash":"hash","date":"2026-01-01T00:00:00Z","data":{"schema":{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{},"queries":{"accounts":{"resource":"accounts","params":{"pageSize":10}}}}}}]}}`
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "preserved")
	clients := &fctl.StackClients{
		URI: "https://stack.example.test",
		HTTPClient: &http.Client{Transport: importRoundTripper(func(request *http.Request) (*http.Response, error) {
			require.Equal(t, "preserved", request.Context().Value(contextKey{}))
			require.Equal(t, http.MethodGet, request.Method)
			require.Equal(t, "/api/ledger/v2/test/logs", request.URL.Path)
			require.Equal(t, "1", request.URL.Query().Get("pageSize"))
			return importLogResponse(http.StatusOK, "application/json", body), nil
		})},
	}
	// This fixture reproduces the generated union's missing-discriminator error.
	sdk := formance.New(formance.WithServerURL(clients.URI), formance.WithClient(clients.HTTPClient))
	pageSize := int64(1)
	_, err := sdk.Ledger.V2.ListLogs(ctx, operations.V2ListLogsRequest{Ledger: "test", PageSize: &pageSize})
	require.Error(t, err)
	id, err := lastImportLogID(ctx, clients, "test")
	require.NoError(t, err)
	require.Equal(t, "9007199254740993", id.String())
}

func TestLastImportLogIDResponseValidation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		status      int
		contentType string
		body        string
		wantID      string
	}{
		{name: "empty", status: 200, contentType: "application/json", body: `{"cursor":{"pageSize":1,"hasMore":false,"data":[]}}`, wantID: "-1"},
		{name: "missing cursor", status: 200, contentType: "application/json", body: `{}`},
		{name: "null data", status: 200, contentType: "application/json", body: `{"cursor":{"pageSize":1,"hasMore":false,"data":null}}`},
		{name: "missing id", status: 200, contentType: "application/json", body: `{"cursor":{"data":[{}]}}`},
		{name: "null id", status: 200, contentType: "application/json", body: `{"cursor":{"data":[{"id":null}]}}`},
		{name: "fractional id", status: 200, contentType: "application/json", body: `{"cursor":{"data":[{"id":1.5}]}}`},
		{name: "malformed", status: 200, contentType: "application/json", body: `{"cursor":`},
		{name: "non JSON", status: 200, contentType: "text/plain", body: `{"cursor":{"data":[{"id":7}]}}`},
		{name: "unauthorized", status: 401, contentType: "application/json", body: `{"cursor":{"data":[{"id":7}]}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clients := &fctl.StackClients{URI: "https://stack.example.test", HTTPClient: &http.Client{
				Transport: importRoundTripper(func(*http.Request) (*http.Response, error) {
					return importLogResponse(test.status, test.contentType, test.body), nil
				}),
			}}
			id, err := lastImportLogID(context.Background(), clients, "test")
			if test.wantID == "" {
				require.Error(t, err)
				require.Nil(t, id)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantID, id.String())
		})
	}
}

func TestLastImportLogIDReadAndTransportErrors(t *testing.T) {
	t.Parallel()
	failure := errors.New("read failed")
	for _, bodyFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(bodyFailure), func(t *testing.T) {
			t.Parallel()
			clients := &fctl.StackClients{URI: "https://stack.example.test", HTTPClient: &http.Client{
				Transport: importRoundTripper(func(*http.Request) (*http.Response, error) {
					if !bodyFailure {
						return nil, failure
					}
					response := importLogResponse(200, "application/json", "")
					response.Body = io.NopCloser(importErrorReader{err: failure})
					return response, nil
				}),
			}}
			id, err := lastImportLogID(context.Background(), clients, "test")
			require.ErrorIs(t, err, failure)
			require.Nil(t, id)
		})
	}
}

type importRoundTripper func(*http.Request) (*http.Response, error)

func (f importRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
func importLogResponse(status int, contentType, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body))}
}

type importErrorReader struct{ err error }

func (r importErrorReader) Read([]byte) (int, error) { return 0, r.err }
