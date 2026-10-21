package payments

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func poolOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("pools create", "create", http.MethodPost, "pools", 0, 0)), "name"),
		commandapi.Leaf("pools get", "get <pool-id>", http.MethodGet, "pools/$0", 1, 1),
		commandapi.Confirmed(commandapi.Leaf("pools delete", "delete <pool-id>", http.MethodDelete, "pools/$0", 1, 1)),
		commandapi.Leaf("pools latest-balances", "latest-balances <pool-id>", http.MethodGet, "pools/$0/balances/latest", 1, 1),
		commandapi.Confirmed(commandapi.Leaf("pools remove-account", "remove-account <pool-id> <account-id>", http.MethodDelete, "pools/$0/accounts/$1", 2, 2)),
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("pools update-query", "update-query <pool-id>", http.MethodPatch, "pools/$0/query", 1, 1)), "query"),
	}
}

func addPoolAccountOperation() commandapi.Operation {
	op := commandapi.Leaf("pools add-account", "add-account <pool-id> <account-id>", http.MethodPost, "pools/$0/accounts", 2, 2)
	op.Body = addPoolAccountBody
	return op
}

func poolBalancesOperation() commandapi.Operation {
	op := commandapi.Leaf("pools balances", "balances <pool-id> <at>", http.MethodGet, "pools/$0/balances", 2, 2)
	op.Body = poolBalancesBody
	return op
}

func addPoolAccountBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := commandapi.Identifier(r.Args[1]); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"accountID": r.Args[1]})
}

func poolBalancesBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	return nil, commandapi.Timestamp(r.Args[1])
}

func validatePoolPayload(object map[string]json.RawMessage, major, minor int) error {

	if raw, exists := object["query"]; exists && string(raw) != "null" {
		if major != 3 || minor < 1 {
			return fmt.Errorf("dynamic pools require Payments >= 3.1.0")
		}
		if _, err := commandapi.ObjectBody(raw); err != nil {
			return fmt.Errorf("query: %w", err)
		}
	}
	if raw, exists := object["accountIDs"]; exists && string(raw) != "null" {
		var ids []string
		if json.Unmarshal(raw, &ids) != nil {
			return fmt.Errorf("accountIDs must be a string array")
		}
		if err := validateIdentifiers(ids); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentifiers(ids []string) error {
	for _, id := range ids {
		if err := commandapi.Identifier(id); err != nil {
			return err
		}
	}
	return nil
}
