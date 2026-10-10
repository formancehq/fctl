package payments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func transferOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("transfer-initiation create", "create", http.MethodPost, "transfer-initiations", 0, 0)), "amount", "asset", "description", "destinationAccountID", "reference", "scheduledAt", "sourceAccountID", "type", "validated"),
		commandapi.Leaf("transfer-initiation get", "get <transfer-id>", http.MethodGet, "transfer-initiations/$0", 1, 1),
		commandapi.Confirmed(commandapi.Leaf("transfer-initiation delete", "delete <transfer-id>", http.MethodDelete, "transfer-initiations/$0", 1, 1)),
		commandapi.Confirmed(commandapi.Leaf("transfer-initiation retry", "retry <transfer-id>", http.MethodPost, "transfer-initiations/$0/retry", 1, 1)),
		commandapi.Payload(commandapi.Confirmed(commandapi.Leaf("transfer-initiation reverse", "reverse <transfer-id>", http.MethodPost, "transfer-initiations/$0/reverse", 1, 1)), "amount", "asset", "description", "metadata", "reference"),
		commandapi.Confirmed(commandapi.Leaf("transfer-initiation approve", "approve <transfer-id>", http.MethodPost, "payment-initiations/$0/approve", 1, 1)),
		commandapi.Confirmed(commandapi.Leaf("transfer-initiation reject", "reject <transfer-id>", http.MethodPost, "payment-initiations/$0/reject", 1, 1)),
	}
}

func transferStatusOperation() commandapi.Operation {
	op := commandapi.Confirmed(commandapi.Leaf("transfer-initiation update-status", "update-status <transfer-id> <status>", http.MethodPost, "transfer-initiations/$0/status", 2, 2))
	op.Body = transferStatusBody
	return op
}

func transferStatusBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if !slices.Contains([]string{"REJECTED", "VALIDATED"}, r.Args[1]) {
		return nil, fmt.Errorf("status must be REJECTED or VALIDATED")
	}
	return json.Marshal(map[string]string{"status": r.Args[1]})
}

// Unversioned SDK V1 routes remain available on Payments 3. Apply version
// prefixes only to operations that the historical CLI delegated to SDK V3.
