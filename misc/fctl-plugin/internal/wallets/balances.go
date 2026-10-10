package wallets

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func balanceOperations() []commandapi.Operation {
	balanceCreate := commandapi.Confirmed(targetWallet(commandapi.Leaf("balances create", "create <balance-name>", http.MethodPost, "wallets/@id/balances", 1, 1, commandapi.StringFlag("expires-at", "RFC3339 expiry"), commandapi.StringFlag("priority", "Integer priority")), true))
	balanceCreate.Body = balanceCreateBody
	balanceList := targetWallet(commandapi.Leaf("balances list", "list", http.MethodGet, "wallets/@id/balances", 0, 0, commandapi.PaginationFlags()...), true)
	balanceList.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
	balanceShow := targetWallet(commandapi.Leaf("balances show", "show <balance-name>", http.MethodGet, "wallets/@id/balances/$0", 1, 1), true)
	return []commandapi.Operation{balanceCreate, balanceList, balanceShow}
}

func balanceCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := commandapi.Identifier(r.Args[0]); err != nil {
		return nil, err
	}
	body := map[string]any{"name": r.Args[0]}
	if v := r.Flags["expires-at"]; v != "" {
		if err := commandapi.Timestamp(v); err != nil {
			return nil, err
		}
		body["expiresAt"] = v
	}
	if v := r.Flags["priority"]; v != "" {
		n, err := integer(v, false)
		if err != nil {
			return nil, err
		}
		body["priority"] = n
	}
	return json.Marshal(body)
}
