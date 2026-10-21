package ledger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEscapesIdentifiersExactlyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/v2/ledger%2F%25/accounts/users%2F%25/metadata/key%2F%25"
		if r.URL.EscapedPath() != want {
			t.Errorf("escaped path %q, want %q", r.URL.EscapedPath(), want)
		}
		w.WriteHeader(204)
	}))
	t.Cleanup(server.Close)
	r := executeRequest("accounts delete-metadata", []string{"users/%", "key/%"}, map[string]string{"ledger": "ledger/%"}, "")
	r.Endpoint = server.URL
	result, err := New(server.Client()).Execute(t.Context(), r)
	if err != nil || string(result.Data) != "null" {
		t.Fatalf("got %s: %v", result.Data, err)
	}
}
