package wallets

import (
	"net/http"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func transactionOperations() []commandapi.Operation {
	transactions := targetWallet(commandapi.Leaf("transactions list", "list", http.MethodGet, "transactions", 0, 0, commandapi.PaginationFlags()...), false)
	transactions.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "id": "walletID"}
	return []commandapi.Operation{transactions}
}
