package payments

import (
	"fmt"
	"net/http"
	"strings"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func paymentListOperations() []commandapi.Operation {
	operations := []commandapi.Operation{}
	for _, pair := range [][2]string{
		{"accounts list", "accounts"}, {"accounts balances", "accounts/$0/balances"}, {"payments list", "payments"},
		{"bank-accounts list", "bank-accounts"}, {"pools list", "pools"}, {"transfer-initiation list", "transfer-initiations"},
		{"connectors list", "connectors"}, {"connectors schedules list", "connectors/$0/schedules"},
		{"connectors schedules instances list", "connectors/$0/schedules/$1/instances"},
	} {
		args := strings.Count(pair[1], "$")
		use := "list"
		if pair[0] == "accounts balances" {
			use = "balances"
		}
		for i := range args {
			use += fmt.Sprintf(" <id-%d>", i+1)
		}
		op := commandapi.Leaf(pair[0], use, http.MethodGet, pair[1], args, args, commandapi.PaginationFlags()...)
		op.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
		if pair[0] == "connectors list" {
			op.Spec.Flags[1].Default = "10"
		}
		operations = append(operations, op)
	}
	return operations
}
