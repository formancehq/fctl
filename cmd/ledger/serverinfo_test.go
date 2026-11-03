package ledger

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
)

func TestServerInfoUsesCurrentLedgerDiscoveryContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/stack/api/ledger/_/info", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"server":"ledger","version":"v3.0.0-rc.2"}`)
	}))
	defer server.Close()
	controller := NewServerInfoController()
	err := controller.readInfo(context.Background(), formance.New(formance.WithServerURL(server.URL+"/stack"), formance.WithClient(server.Client())))
	require.NoError(t, err)
	require.Equal(t, &ServerInfoStore{Server: "ledger", Version: "v3.0.0-rc.2"}, controller.GetStore())
}
