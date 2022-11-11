package internal

import (
	"context"
	"errors"
	"strconv"

	ledgerclient "github.com/numary/ledger/client"
)

func TransactionIDOrLast(ctx context.Context, ledgerClient *ledgerclient.APIClient, ledger, id string) (int64, error) {
	if id == "last" {
		response, _, err := ledgerClient.TransactionsApi.
			ListTransactions(ctx, ledger).
			PageSize(1).
			Execute()
		if err != nil {
			return 0, err
		}
		if len(response.Cursor.Data) == 0 {
			return 0, errors.New("no transaction found")
		}
		return int64(response.Cursor.Data[0].Txid), nil
	}

	return strconv.ParseInt(id, 10, 32)
}
