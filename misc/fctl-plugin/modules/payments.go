package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func newPayments(client *http.Client) pluginsdk.Plugin {
	operations := []operation{
		payload(confirmed(leaf("accounts create", "create", http.MethodPost, "accounts", 0, 0)), "connectorID", "createdAt", "reference", "type"),
		leaf("accounts get", "get <account-id>", http.MethodGet, "accounts/$0", 1, 1),
		payload(confirmed(leaf("payments create", "create", http.MethodPost, "payments", 0, 0)), "amount", "asset", "connectorID", "createdAt", "reference", "scheme", "status", "type"),
		leaf("payments get", "get <payment-id>", http.MethodGet, "payments/$0", 1, 1),
		payload(confirmed(leaf("bank-accounts create", "create", http.MethodPost, "bank-accounts", 0, 0)), "name"),
		leaf("bank-accounts get", "get <bank-account-id>", http.MethodGet, "bank-accounts/$0", 1, 1),
		payload(confirmed(leaf("pools create", "create", http.MethodPost, "pools", 0, 0)), "name"),
		leaf("pools get", "get <pool-id>", http.MethodGet, "pools/$0", 1, 1),
		confirmed(leaf("pools delete", "delete <pool-id>", http.MethodDelete, "pools/$0", 1, 1)),
		leaf("pools latest-balances", "latest-balances <pool-id>", http.MethodGet, "pools/$0/balances/latest", 1, 1),
		confirmed(leaf("pools remove-account", "remove-account <pool-id> <account-id>", http.MethodDelete, "pools/$0/accounts/$1", 2, 2)),
		payload(confirmed(leaf("pools update-query", "update-query <pool-id>", http.MethodPatch, "pools/$0/query", 1, 1)), "query"),
		payload(confirmed(leaf("transfer-initiation create", "create", http.MethodPost, "transfer-initiations", 0, 0)), "amount", "asset", "description", "destinationAccountID", "reference", "scheduledAt", "sourceAccountID", "type", "validated"),
		leaf("transfer-initiation get", "get <transfer-id>", http.MethodGet, "transfer-initiations/$0", 1, 1),
		confirmed(leaf("transfer-initiation delete", "delete <transfer-id>", http.MethodDelete, "transfer-initiations/$0", 1, 1)),
		confirmed(leaf("transfer-initiation retry", "retry <transfer-id>", http.MethodPost, "transfer-initiations/$0/retry", 1, 1)),
		payload(confirmed(leaf("transfer-initiation reverse", "reverse <transfer-id>", http.MethodPost, "transfer-initiations/$0/reverse", 1, 1)), "amount", "asset", "description", "metadata", "reference"),
		confirmed(leaf("transfer-initiation approve", "approve <transfer-id>", http.MethodPost, "payment-initiations/$0/approve", 1, 1)),
		confirmed(leaf("transfer-initiation reject", "reject <transfer-id>", http.MethodPost, "payment-initiations/$0/reject", 1, 1)),
		leaf("tasks get", "get <task-id>", http.MethodGet, "tasks/$0", 1, 1),
		leaf("connectors list-available", "list-available", http.MethodGet, "connectors/configs", 0, 0),
		confirmed(payload(leaf("connectors install", "install <connector>", http.MethodPost, "connectors/$0", 1, 1))),
		confirmed(payload(leaf("connectors update-config", "update-config <connector>", http.MethodPost, "connectors/$0/config", 1, 1, str("connector-id", "Connector ID")))),
		leaf("connectors get-config", "get-config", http.MethodGet, "connectors", 0, 0, str("provider", "Legacy provider name"), str("connector-id", "Connector ID")),
		confirmed(leaf("connectors uninstall", "uninstall", http.MethodDelete, "connectors", 0, 0, str("provider", "Legacy provider name"), str("connector-id", "Connector ID"))),
		leaf("connectors schedules get", "get <connector-id> <schedule-id>", http.MethodGet, "connectors/$0/schedules/$1", 2, 2),
		leaf("orders get", "get <order-id>", http.MethodGet, "orders/$0", 1, 1),
		leaf("conversions get", "get <conversion-id>", http.MethodGet, "conversions/$0", 1, 1),
	}
	operations = append(operations, paymentListOperations()...)
	operations = append(operations, paymentExchangeOperations()...)
	operations = append(operations, paymentMetadataOperations()...)
	forward := confirmed(leaf("bank-accounts forward", "forward <bank-account-id> <connector-id>", http.MethodPost, "bank-accounts/$0/forward", 2, 2))
	forward.body = forwardBankAccountBody
	add := leaf("pools add-account", "add-account <pool-id> <account-id>", http.MethodPost, "pools/$0/accounts", 2, 2)
	add.body = addPoolAccountBody
	balances := leaf("pools balances", "balances <pool-id> <at>", http.MethodGet, "pools/$0/balances", 2, 2)
	balances.body = poolBalancesBody
	status := confirmed(leaf("transfer-initiation update-status", "update-status <transfer-id> <status>", http.MethodPost, "transfer-initiations/$0/status", 2, 2))
	status.body = transferStatusBody
	operations = append(operations, forward, add, balances, status)
	for i := range operations {
		operations[i].run = paymentsOperation
		if operations[i].command == "connectors get-config" || operations[i].command == "connectors uninstall" || operations[i].command == "connectors update-config" {
			alternative := "provider"
			if operations[i].command == "connectors update-config" {
				alternative = ""
			}
			operations[i].spec.Inputs = append(operations[i].spec.Inputs, pluginsdk.InputSpec{Title: "Connector", Kind: "select", Flag: "connector-id", AlternativeFlag: alternative, Required: true,
				Source: &pluginsdk.ChoiceSource{CommandPath: []string{"payments", "connectors", "list"}, ValueField: "id", LabelFields: []string{"name", "provider", "id"}, EmptyMessage: "No connectors available"}})
		}
	}
	return newPlugin("payments", client, operations)
}

func paymentMetadata(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if len(r.Args) > 1 && r.Flags["metadata"] != "" {
		return nil, fmt.Errorf("provide metadata once")
	}
	var metadata map[string]string
	if len(r.Args) > 1 {
		var err error
		metadata, err = positionalMetadata(r.Args[1:])
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		metadata, err = pairs(r.Flags["metadata"])
		if err != nil {
			return nil, err
		}
	}
	if len(metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}
	if r.CommandPath[1] == "bank-accounts" {
		return marshal(map[string]any{"metadata": metadata})
	}
	return marshal(metadata)
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
	return marshal(map[string]any{"$and": matches})
}

var paymentSemver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:[-+][0-9A-Za-z.+-]+)?$`)
var paymentCommit = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

func paymentsVersion(ctx context.Context, client *httpclient.Client) (major, minor int, err error) {
	info, err := client.Do(ctx, http.MethodGet, httpclient.Path("_info"), nil, nil, nil)
	if err != nil {
		return 0, 0, err
	}
	version, err := pluginsdk.ServiceVersion(info)
	if err != nil {
		return 0, 0, err
	}
	match := paymentSemver.FindStringSubmatch(version)
	if match == nil {
		if paymentCommit.MatchString(version) {
			return 3, int(^uint(0) >> 1), nil
		}
		return 0, 0, fmt.Errorf("unrecognized Payments version %q", version)
	}
	major, err = strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, err
	}
	if major > 3 {
		return 0, 0, fmt.Errorf("unsupported Payments major %d", major)
	}
	return major, minor, nil
}

func paymentsOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	major, minor, err := paymentsVersion(ctx, client)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(op.command, "connectors ") {
		return paymentConnector(ctx, client, r, op, body, major)
	}
	v3, err := paymentV3Route(op.command, major, minor)
	if err != nil {
		return nil, err
	}
	if err := validatePaymentPayload(op.command, body, major, minor); err != nil {
		return nil, err
	}
	if v3 {
		op.segments = append([]string{"v3"}, op.segments...)
	}
	query, err := queryValues(op.query, r)
	if err != nil {
		return nil, err
	}
	if op.command == "pools balances" {
		query.Set("at", r.Args[1])
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
		body = nil
	}
	path, err := route(op.segments, r)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, op.method, path, query, body, nil)
}

func validatePaymentPayload(command string, body json.RawMessage, major, minor int) error {
	if body == nil {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return fmt.Errorf("expected a JSON object")
	}
	for _, validate := range []func(map[string]json.RawMessage) error{validatePaymentNumbers, validatePaymentTimes, validatePaymentStrings, validatePaymentMetadata} {
		if err := validate(object); err != nil {
			return err
		}
	}
	if err := validatePaymentEnums(command, object); err != nil {
		return err
	}
	if command == "bank-accounts create" && major < 3 {
		if _, err := objectBody(body, "country"); err != nil {
			return err
		}
	}
	if command == "pools create" || command == "pools update-query" {
		return validatePoolPayload(object, major, minor)
	}
	return nil
}

func rawEnum(object map[string]json.RawMessage, field string, allowed []string) error {
	var value string
	if json.Unmarshal(object[field], &value) != nil || !slices.Contains(allowed, value) {
		return fmt.Errorf("invalid payload %s", field)
	}
	return nil
}

func paymentListOperations() []operation {
	operations := []operation{}
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
		op := leaf(pair[0], use, http.MethodGet, pair[1], args, args, pagination()...)
		op.query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
		if pair[0] == "connectors list" {
			op.spec.Flags[1].Default = "10"
		}
		operations = append(operations, op)
	}
	return operations
}

func paymentExchangeOperations() []operation {
	operations := []operation{}
	for _, group := range []string{"orders", "conversions"} {
		flags := append(pagination(), str("connector-id", "Connector ID filter"), str("reference", "Provider reference filter"),
			str("status", "Status filter"), str("source-asset", "Source asset filter"), str("destination-asset", "Destination asset filter"))
		if group == "orders" {
			flags = append(flags, str("direction", "BUY, SELL or UNKNOWN"), str("type", "Order type"))
		}
		op := leaf(group+" list", "list", http.MethodGet, group, 0, 0, flags...)
		op.query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
		op.body = exchangeQuery
		operations = append(operations, op)
	}
	return operations
}

func paymentMetadataOperations() []operation {
	operations := []operation{}
	for _, group := range []string{"payments", "bank-accounts"} {
		command, path := "set-metadata", "payments/$0/metadata"
		if group == "bank-accounts" {
			command, path = "update-metadata", "bank-accounts/$0/metadata"
		}
		op := confirmed(leaf(group+" "+command, command+" <id> [key=value...]", http.MethodPatch, path, 1, 1000, metadataFlag()))
		op.body = paymentMetadata
		operations = append(operations, op)
	}
	return operations
}

func forwardBankAccountBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := identifier(r.Args[1]); err != nil {
		return nil, err
	}
	return marshal(map[string]string{"connectorID": r.Args[1]})
}

func addPoolAccountBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := identifier(r.Args[1]); err != nil {
		return nil, err
	}
	return marshal(map[string]string{"accountID": r.Args[1]})
}

func poolBalancesBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	return nil, timestamp(r.Args[1])
}

func transferStatusBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if !slices.Contains([]string{"REJECTED", "VALIDATED"}, r.Args[1]) {
		return nil, fmt.Errorf("status must be REJECTED or VALIDATED")
	}
	return marshal(map[string]string{"status": r.Args[1]})
}

// Unversioned SDK V1 routes remain available on Payments 3. Apply version
// prefixes only to operations that the historical CLI delegated to SDK V3.
func paymentV3Route(command string, major, minor int) (bool, error) {
	if strings.HasPrefix(command, "bank-accounts ") || command == "pools create" || command == "pools latest-balances" {
		return major == 3, nil
	}
	if strings.HasPrefix(command, "orders ") || strings.HasPrefix(command, "conversions ") {
		if major != 3 || minor < 3 {
			return false, fmt.Errorf("%s requires Payments >= 3.3.0", strings.Fields(command)[0])
		}
		return true, nil
	}
	switch command {
	case "tasks get", "transfer-initiation approve", "transfer-initiation reject":
		if major != 3 {
			return false, fmt.Errorf("%s requires Payments >= 3.0.0", command)
		}
		return true, nil
	case "pools update-query":
		if major != 3 || minor < 1 {
			return false, fmt.Errorf("update-query requires Payments >= 3.1.0")
		}
		return true, nil
	case "transfer-initiation update-status":
		if major == 3 {
			return false, fmt.Errorf("update-status is unavailable on Payments 3; use approve or reject")
		}
	}
	return false, nil
}

func validatePaymentNumbers(object map[string]json.RawMessage) error {
	for _, field := range []string{"amount", "initialAmount"} {
		if raw, exists := object[field]; exists {
			if _, err := integer(string(raw), true); err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
		}
	}
	return nil
}

func validatePaymentTimes(object map[string]json.RawMessage) error {
	for _, field := range []string{"createdAt", "scheduledAt"} {
		if raw, exists := object[field]; exists {
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return fmt.Errorf("%s must be an RFC3339 string", field)
			}
			if err := timestamp(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePaymentStrings(object map[string]json.RawMessage) error {
	for _, field := range []string{"name", "connectorID", "reference", "asset", "description", "sourceAccountID", "destinationAccountID", "scheme"} {
		if raw, exists := object[field]; exists && string(raw) != "null" {
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return fmt.Errorf("%s must be a string", field)
			}
		}
	}
	return nil
}

func validatePaymentMetadata(object map[string]json.RawMessage) error {
	if raw, exists := object["metadata"]; exists && string(raw) != "null" {
		var values map[string]string
		if json.Unmarshal(raw, &values) != nil || values == nil {
			return fmt.Errorf("metadata must be an object with string values")
		}
	}
	if raw, exists := object["validated"]; exists {
		var value bool
		if json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("validated must be a boolean")
		}
	}
	return nil
}

func validatePaymentEnums(command string, object map[string]json.RawMessage) error {
	if command == "accounts create" {
		if err := rawEnum(object, "type", []string{"UNKNOWN", "INTERNAL", "EXTERNAL"}); err != nil {
			return err
		}
	}
	if command == "payments create" {
		if err := rawEnum(object, "type", []string{"PAY-IN", "PAYOUT", "TRANSFER", "OTHER"}); err != nil {
			return err
		}
		if err := rawEnum(object, "status", []string{"PENDING", "SUCCEEDED", "CANCELLED", "FAILED", "EXPIRED", "REFUNDED", "REFUNDED_FAILURE", "DISPUTE", "DISPUTE_WON", "DISPUTE_LOST", "OTHER"}); err != nil {
			return err
		}
	}
	if command == "transfer-initiation create" {
		if err := rawEnum(object, "type", []string{"TRANSFER", "PAYOUT"}); err != nil {
			return err
		}
	}
	return nil
}

func validatePoolPayload(object map[string]json.RawMessage, major, minor int) error {

	if raw, exists := object["query"]; exists && string(raw) != "null" {
		if major != 3 || minor < 1 {
			return fmt.Errorf("dynamic pools require Payments >= 3.1.0")
		}
		if _, err := objectBody(raw); err != nil {
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

func positionalMetadata(args []string) (map[string]string, error) {
	metadata := map[string]string{}
	for _, v := range args {
		key, value, ok := strings.Cut(v, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected metadata key=value")
		}
		if _, duplicate := metadata[key]; duplicate {
			return nil, fmt.Errorf("duplicate metadata key %q", key)
		}
		metadata[key] = value
	}
	return metadata, nil
}

func validateIdentifiers(ids []string) error {
	for _, id := range ids {
		if err := identifier(id); err != nil {
			return err
		}
	}
	return nil
}
