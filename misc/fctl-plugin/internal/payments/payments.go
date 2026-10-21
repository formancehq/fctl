package payments

import (
	"net/http"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func paymentOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("payments create", "create", http.MethodPost, "payments", 0, 0)), "amount", "asset", "connectorID", "createdAt", "reference", "scheme", "status", "type"),
		commandapi.Leaf("payments get", "get <payment-id>", http.MethodGet, "payments/$0", 1, 1),
	}
}
