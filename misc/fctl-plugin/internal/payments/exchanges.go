package payments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func orderOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Leaf("orders get", "get <order-id>", http.MethodGet, "orders/$0", 1, 1),
	}
}

func conversionOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Leaf("conversions get", "get <conversion-id>", http.MethodGet, "conversions/$0", 1, 1),
	}
}

func exchangeQuery(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	group := r.CommandPath[1]
	allowed := map[string][]string{"direction": {"BUY", "SELL", "UNKNOWN"}}
	if group == "orders" {
		allowed["status"] = []string{"PENDING", "OPEN", "PARTIALLY_FILLED", "FILLED", "CANCELLED", "FAILED", "EXPIRED", "UNKNOWN"}
		allowed["type"] = []string{"MARKET", "LIMIT", "STOP", "STOP_LIMIT", "TWAP", "VWAP", "PEG", "BLOCK", "RFQ", "TRAILING_STOP", "TRAILING_STOP_LIMIT", "TAKE_PROFIT", "TAKE_PROFIT_LIMIT", "LIMIT_MAKER", "UNKNOWN"}
	} else {
		allowed["status"] = []string{"PENDING", "COMPLETED", "FAILED", "UNKNOWN"}
	}
	for flag, values := range allowed {
		if v := r.Flags[flag]; v != "" && !slices.Contains(values, v) {
			return nil, fmt.Errorf("invalid --%s %q", flag, v)
		}
	}
	if r.Flags["cursor"] != "" {
		return nil, nil
	}
	matches := []any{}
	for _, flag := range []string{"connector-id", "reference", "direction", "status", "type", "source-asset", "destination-asset"} {
		if value := r.Flags[flag]; value != "" {
			matches = append(matches, map[string]any{"$match": map[string]string{strings.ReplaceAll(flag, "-", "_"): value}})
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	return json.Marshal(map[string]any{"$and": matches})
}

func paymentExchangeOperations() []commandapi.Operation {
	operations := []commandapi.Operation{}
	for _, group := range []string{"orders", "conversions"} {
		flags := append(commandapi.PaginationFlags(), commandapi.StringFlag("connector-id", "Connector ID filter"), commandapi.StringFlag("reference", "Provider reference filter"),
			commandapi.StringFlag("status", "Status filter"), commandapi.StringFlag("source-asset", "Source asset filter"), commandapi.StringFlag("destination-asset", "Destination asset filter"))
		if group == "orders" {
			flags = append(flags, commandapi.StringFlag("direction", "BUY, SELL or UNKNOWN"), commandapi.StringFlag("type", "Order type"))
		}
		op := commandapi.Leaf(group+" list", "list", http.MethodGet, group, 0, 0, flags...)
		op.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
		op.Body = exchangeQuery
		operations = append(operations, op)
	}
	return operations
}
