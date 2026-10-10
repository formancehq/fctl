package wallets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func movementOperations() []commandapi.Operation {
	credit := commandapi.Confirmed(targetWallet(commandapi.Leaf("credit", "credit <amount> <asset>", http.MethodPost, "wallets/@id/credit", 2, 2,
		metadataFlag(), ikFlag(), commandapi.StringFlag("balance", "Balance to credit"), commandapi.StringFlag("source", "Sources as CSV or JSON string array: account=ADDRESS or wallet=id:ID/BALANCE or wallet=name:NAME/BALANCE")), true))
	credit.Body = walletMovement
	debit := commandapi.Confirmed(targetWallet(commandapi.Leaf("debit", "debit <amount> <asset>", http.MethodPost, "wallets/@id/debit", 2, 2,
		metadataFlag(), ikFlag(), commandapi.StringFlag("balance", "Balances as CSV or JSON string array"), commandapi.BoolFlag("pending", "Create a pending hold"), commandapi.StringFlag("description", "Debit description"),
		commandapi.StringFlag("destination", "account=ADDRESS or wallet=id:ID/BALANCE or wallet=name:NAME/BALANCE")), true))
	debit.Body = walletMovement
	return []commandapi.Operation{credit, debit}
}

func walletMovement(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	n, err := integer(r.Args[0], true)
	if err != nil {
		return nil, err
	}
	if err := commandapi.Identifier(r.Args[1]); err != nil {
		return nil, fmt.Errorf("asset: %w", err)
	}
	metadata, err := commandapi.Pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{"amount": map[string]any{"amount": n, "asset": r.Args[1]}, "metadata": metadata}
	if r.CommandPath[1] == "credit" {
		body["balance"] = r.Flags["balance"]
		sources, err := commandapi.List(r.Flags["source"])
		if err != nil {
			return nil, err
		}
		if err := validateSubjects(sources); err != nil {
			return nil, err
		}
	} else {
		body["pending"] = r.Flags["pending"] == "true"
		body["description"] = r.Flags["description"]
		balances, err := commandapi.List(r.Flags["balance"])
		if err != nil {
			return nil, err
		}
		body["balances"] = balances
		if value := r.Flags["destination"]; value != "" {
			if _, err := parseSubject(value); err != nil {
				return nil, err
			}
		}
	}
	return json.Marshal(body)
}
