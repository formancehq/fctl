package payments

import (
	"net/http"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func accountOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("accounts create", "create", http.MethodPost, "accounts", 0, 0)), "connectorID", "createdAt", "reference", "type"),
		commandapi.Leaf("accounts get", "get <account-id>", http.MethodGet, "accounts/$0", 1, 1),
	}
}
