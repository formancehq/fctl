package payments

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func bankAccountOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("bank-accounts create", "create", http.MethodPost, "bank-accounts", 0, 0)), "name"),
		commandapi.Leaf("bank-accounts get", "get <bank-account-id>", http.MethodGet, "bank-accounts/$0", 1, 1),
	}
}

func forwardBankAccountOperation() commandapi.Operation {
	op := commandapi.Confirmed(commandapi.Leaf("bank-accounts forward", "forward <bank-account-id> <connector-id>", http.MethodPost, "bank-accounts/$0/forward", 2, 2))
	op.Body = forwardBankAccountBody
	return op
}

func forwardBankAccountBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := commandapi.Identifier(r.Args[1]); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"connectorID": r.Args[1]})
}
