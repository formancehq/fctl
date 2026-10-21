package wallets

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func holdOperations() []commandapi.Operation {
	holds := targetWallet(commandapi.Leaf("holds list", "list", http.MethodGet, "holds", 0, 0, append(commandapi.PaginationFlags(), metadataFlag())...), false)
	holds.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "id": "walletID"}
	holds.Body = validateMetadata
	confirm := commandapi.Confirmed(commandapi.Leaf("holds confirm", "confirm <hold-id>", http.MethodPost, "holds/$0/confirm", 1, 1, commandapi.BoolFlag("final", "Close hold after confirming"), commandapi.StringFlag("amount", "Integer amount (default 0)"), ikFlag()))
	confirm.Body = confirmHoldBody
	return []commandapi.Operation{holds, confirm,
		commandapi.Leaf("holds show", "show <hold-id>", http.MethodGet, "holds/$0", 1, 1),
		commandapi.Confirmed(commandapi.Leaf("holds void", "void <hold-id>", http.MethodPost, "holds/$0/void", 1, 1, ikFlag())),
	}
}

func confirmHoldBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	amount := r.Flags["amount"]
	if amount == "" {
		amount = "0"
	}
	n, err := integer(amount, true)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"amount": n, "final": r.Flags["final"] == "true"})
}
