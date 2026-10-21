package ledger

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func TestStructuredErrorsAndNoMutationRetry(t *testing.T) {
	var calls atomic.Int32
	want := `{"errorCode":"CONFLICT","errorMessage":"duplicate","details":{"id":` + huge + `}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(409); writeJSON(t, w, want) }))
	t.Cleanup(server.Close)
	r := executeRequest("send", []string{"a", "1", "USD"}, nil, "")
	r.Endpoint = server.URL
	data, err := New(server.Client()).Execute(t.Context(), r)
	var failure *httpclient.Error
	if !errors.As(err, &failure) || failure.StatusCode != 409 || failure.Code != "CONFLICT" || string(data.Data) != want || calls.Load() != 1 {
		t.Fatalf("data %s, calls %d, error %v", data.Data, calls.Load(), err)
	}
}

//nolint:gocognit // The matrix checks the two-request lastN contract for each supported operation.
func TestTransactionLastNPreservesBigInteger(t *testing.T) {
	for _, command := range []string{"transactions show", "transactions revert", "transactions set-metadata", "transactions delete-metadata"} {
		t.Run(command, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					if r.Method != "GET" || r.URL.Path != "/default/transactions" || r.URL.Query().Get("pageSize") != "1" {
						t.Errorf("unexpected lastN lookup: %s %s", r.Method, r.URL)
					}
					writeJSON(t, w, `{"cursor":{"data":[{"txid":`+huge+`}]}}`)
					return
				}
				expected := "1234567890123456789012345678901234567888"
				if !strings.Contains(r.URL.Path, "/transactions/"+expected) {
					t.Errorf("lost ID precision: %s", r.URL)
				}
				writeJSON(t, w, `{"data":{"txid":`+expected+`}}`)
			}))
			t.Cleanup(server.Close)
			args := []string{"last2"}
			if strings.Contains(command, "metadata") {
				args = append(args, "key=value")
			}
			r := executeRequest(command, args, nil, "")
			r.Endpoint = server.URL
			if _, err := New(server.Client()).Execute(t.Context(), r); err != nil || calls.Load() != 2 {
				t.Fatalf("calls %d: %v", calls.Load(), err)
			}
		})
	}
}

func TestTransactionLastWithoutOffsetAndBadResults(t *testing.T) {
	for _, data := range []string{`{"cursor":{"data":[]}}`, `{"cursor":{"data":[{"txid":1}]}}`, `{"cursor":{"data":[{"txid":"bad"}]}}`} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); writeJSON(t, w, data) }))
		r := executeRequest("transactions show", []string{"last2"}, nil, "")
		r.Endpoint = server.URL
		if _, err := New(server.Client()).Execute(t.Context(), r); err == nil || calls.Load() != 1 {
			t.Errorf("bad lastN result %s, calls %d, error %v", data, calls.Load(), err)
		}
		server.Close()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/default/transactions" {
			writeJSON(t, w, `{"cursor":{"data":[{"txid":`+huge+`}]}}`)
			return
		}
		if r.URL.Path != "/default/transactions/"+huge {
			t.Errorf("last path %s", r.URL)
		}
		writeJSON(t, w, `{"data":{}}`)
	}))
	t.Cleanup(server.Close)
	r := executeRequest("transactions show", []string{"last"}, nil, "")
	r.Endpoint = server.URL
	if _, err := New(server.Client()).Execute(t.Context(), r); err != nil {
		t.Fatal(err)
	}
}

func TestNumscriptVariablesAndExplicitClears(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		assertJSON(t, payload, []byte(`{"script":{"plain":"script","vars":{"account":"users:1","portion":"1/2","amount":{"amount":`+huge+`,"asset":"USD/2"},"raw":`+huge+`}},"reference":"","metadata":{},"future":{"id":`+huge+`}}`))
		writeJSON(t, w, `{"data":[]}`)
	}))
	t.Cleanup(server.Close)
	r := executeRequest("transactions num", []string{"host-read.num"}, map[string]string{"account-var": "account=users:1", "portion-var": "portion=1/2", "amount-var": "amount=" + huge + "/USD/2"}, `{"script":{"plain":"script","vars":{"raw":`+huge+`}},"reference":"old","metadata":{"a":"b"},"timestamp":"2026-10-01T00:00:00Z","future":{"id":`+huge+`}}`)
	r.Flags["reference"], r.Flags["metadata"], r.Flags["timestamp"] = "", "", ""
	r.ChangedFlags = map[string]bool{"reference": true, "metadata": true, "timestamp": true}
	r.Endpoint = server.URL
	if _, err := New(server.Client()).Execute(t.Context(), r); err != nil {
		t.Fatal(err)
	}
}

func TestScriptObjectAndPairFormats(t *testing.T) {
	r := executeRequest("transactions num", nil, map[string]string{"reference": "ref", "timestamp": "2026-10-01T00:00:00Z", "metadata": "key=value"}, `{"plain":"script","vars":{"raw":`+huge+`}}`)
	normalized, err := pluginsdk.NormalizeRequest(manifest(), r)
	if err != nil {
		t.Fatal(err)
	}
	body, err := numPayload(normalized)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, body, []byte(`{"script":{"plain":"script","vars":{"raw":`+huge+`}},"reference":"ref","timestamp":"2026-10-01T00:00:00Z","metadata":{"key":"value"}}`))
	for _, value := range []string{`"a=hello, world",b=a=b`, `["a=hello, world","b=a=b"]`} {
		pairs, err := flagPairs(value)
		if err != nil || pairs["a"] != "hello, world" || pairs["b"] != "a=b" {
			t.Errorf("pairs %v: %v", pairs, err)
		}
	}
	for _, value := range []string{"a=b\nc=d", "=empty-key", "[invalid", `"unclosed`} {
		if _, err := flagPairs(value); err == nil {
			t.Errorf("expected invalid pairs: %s", value)
		}
	}
}
