package reconciliation

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func policyOperations() []command.Operation {
	reconcile := command.Leaf("policies reconcile", "reconcile <policy-id> <at-ledger> <at-payments>", http.MethodPost, "policies/$0/reconciliation", 3, 3)
	reconcile.Body = reconcileBody
	return []command.Operation{
		command.Leaf("policies get", "get <policy-id>", http.MethodGet, "policies/$0", 1, 1),
		command.Confirmed(command.Payload(command.Leaf("policies create", "create", http.MethodPost, "policies", 0, 0), "name", "ledgerName", "paymentsPoolID", "ledgerQuery")),
		command.Confirmed(command.Leaf("policies delete", "delete <policy-id>", http.MethodDelete, "policies/$0", 1, 1)),
		paginatedList("policies list", "list", "policies", 0),
		reconcile,
	}
}

func reconcileBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	for _, value := range r.Args[1:] {
		if err := command.Timestamp(value); err != nil {
			return nil, err
		}
	}
	return json.Marshal(map[string]string{"reconciledAtLedger": r.Args[1], "reconciledAtPayments": r.Args[2]})
}
