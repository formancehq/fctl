package clientcontract

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func TestLedgerV1TransactionRoutes(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()
	sdk := formance.New(formance.WithServerURL(server.URL), formance.WithClient(server.Client()))
	ctx := context.Background()
	_, err := sdk.Ledger.V1.CreateTransaction(ctx, operations.CreateTransactionRequest{Ledger: "demo"})
	require.Error(t, err)
	require.Equal(t, "POST /api/ledger/demo/transactions", method+" "+path)
	_, err = sdk.Ledger.V1.GetTransaction(ctx, operations.GetTransactionRequest{Ledger: "demo", Txid: big.NewInt(42)})
	require.Error(t, err)
	require.Equal(t, "GET /api/ledger/demo/transactions/42", method+" "+path)
	_, err = sdk.Ledger.V1.RevertTransaction(ctx, operations.RevertTransactionRequest{Ledger: "demo", Txid: big.NewInt(42)})
	require.Error(t, err)
	require.Equal(t, "POST /api/ledger/demo/transactions/42/revert", method+" "+path)
}
