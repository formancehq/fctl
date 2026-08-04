package clarity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestClientDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/reconciliation/rules" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("pageSize"); got != "42" {
			t.Errorf("pageSize = %q", got)
		}
		var query map[string]any
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := query["$match"]; !ok {
			t.Errorf("body = %#v", query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cursor":{"pageSize":42,"hasMore":false,"data":[]}}`))
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), BaseURL: server.URL + "/"}
	var response CursorResponse[Rule]
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := client.Do(cmd, http.MethodGet, "/rules", url.Values{"pageSize": {"42"}}, map[string]any{
		"$match": map[string]any{"enabled": true},
	}, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Cursor.PageSize != 42 {
		t.Fatalf("page size = %d", response.Cursor.PageSize)
	}
}

func TestClientDoReturnsStructuredAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errorCode":"VALIDATION","errorMessage":"bad rule","details":"unknown template"}`))
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), BaseURL: server.URL}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := client.Do(cmd, http.MethodPost, "/rules", nil, map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), "VALIDATION") || !strings.Contains(err.Error(), "unknown template") {
		t.Fatalf("error = %v", err)
	}
}

func TestResourcePathEscapesPathSeparators(t *testing.T) {
	t.Parallel()
	if got := ResourcePath("rule/one"); got != "rule%2Fone" {
		t.Fatalf("ResourcePath() = %q", got)
	}
}
