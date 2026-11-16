package demo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectLedgerMajor(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"v3 flat", `{"version":"3.0.0-rc.3","protocolVersion":"40"}`, "v3", 200},
		{"v2 envelope", `{"data":{"server":"ledger","version":"v2.4.0"}}`, "v2", 200},
		{"future", `{"version":"4.0.0"}`, "", 200},
		{"missing version", `{}`, "", 200},
		{"invalid JSON", `not json`, "", 200},
		{"missing route", `{"errorCode":"ROUTE_NOT_FOUND"}`, "", 404},
		{"not authorized", `{}`, "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/prefix/api/ledger/_info", r.URL.Path)
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, "%s", tc.body)
			}))
			defer server.Close()
			major, err := detectLedgerMajor(t.Context(), server.Client(), server.URL+"/prefix/")
			require.Equal(t, 1, calls)
			if tc.want == "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, major)
			}
		})
	}
}
