package ledger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//nolint:gocognit // Each cursor endpoint has distinct historical query defaults.
func TestPaginationMaintainsHistoricalDefaults(t *testing.T) {
	for _, command := range []string{"list", "accounts list", "transactions list", "schemas list", "volumes list"} {
		t.Run(command, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cursor") != "next+opaque" {
					t.Errorf("missing cursor: %s", r.URL)
				}
				wantPage := ""
				switch command {
				case "schemas list":
					wantPage = "15"
				case "volumes list":
					wantPage = "10"
				}
				if r.URL.Query().Get("pageSize") != wantPage {
					t.Errorf("cursor page size %s, want %s", r.URL, wantPage)
				}
				writeJSON(t, w, `{"cursor":{"hasMore":true,"next":"next+2","data":[{"id":`+huge+`}]}}`)
			}))
			t.Cleanup(server.Close)
			r := executeRequest(command, nil, map[string]string{"cursor": "next+opaque"}, "")
			r.Endpoint = server.URL
			data, err := New(server.Client()).Execute(t.Context(), r)
			if err != nil || !strings.Contains(string(data.Data), huge) {
				t.Fatalf("got %s: %v", data.Data, err)
			}
		})
	}
}

func TestServerInfoPreservesNestedAndDirectEnvelopes(t *testing.T) {
	for _, data := range []string{`{"data":{"server":"ledger","version":"v2.4.15","future":true}}`, `{"server":"ledger","version":"v2.4.15"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeJSON(t, w, data) }))
		r := executeRequest("server-infos", nil, nil, "")
		r.Endpoint = server.URL
		result, err := New(server.Client()).Execute(t.Context(), r)
		if err != nil || string(result.Data) != data {
			t.Errorf("info %s: %v", result.Data, err)
		}
		server.Close()
	}
}
