package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func connectorOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Leaf("connectors list-available", "list-available", http.MethodGet, "connectors/configs", 0, 0),
		commandapi.Confirmed(commandapi.Payload(commandapi.Leaf("connectors install", "install <connector>", http.MethodPost, "connectors/$0", 1, 1))),
		commandapi.Confirmed(commandapi.Payload(commandapi.Leaf("connectors update-config", "update-config <connector>", http.MethodPost, "connectors/$0/config", 1, 1, commandapi.StringFlag("connector-id", "Connector ID")))),
		commandapi.Leaf("connectors get-config", "get-config", http.MethodGet, "connectors", 0, 0, commandapi.StringFlag("provider", "Legacy provider name"), commandapi.StringFlag("connector-id", "Connector ID")),
		commandapi.Confirmed(commandapi.Leaf("connectors uninstall", "uninstall", http.MethodDelete, "connectors", 0, 0, commandapi.StringFlag("provider", "Legacy provider name"), commandapi.StringFlag("connector-id", "Connector ID"))),
		commandapi.Leaf("connectors schedules get", "get <connector-id> <schedule-id>", http.MethodGet, "connectors/$0/schedules/$1", 2, 2),
	}
}

func legacyProvider(value string) (string, error) {
	name := strings.ToLower(strings.ReplaceAll(value, "-", ""))
	providers := map[string]string{"adyen": "ADYEN", "atlar": "ATLAR", "bankingcircle": "BANKING-CIRCLE", "currencycloud": "CURRENCY-CLOUD", "mangopay": "MANGOPAY", "modulr": "MODULR", "moneycorp": "MONEYCORP", "stripe": "STRIPE", "wise": "WISE", "generic": "GENERIC", "dummypay": "DUMMY-PAY"}
	if provider := providers[name]; provider != "" {
		return provider, nil
	}
	return "", fmt.Errorf("provider %q is unsupported by the legacy Payments API", value)
}

func paymentConnector(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, body json.RawMessage, major int) (json.RawMessage, error) {
	query, err := commandapi.QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(op.Command, "connectors schedules ") {
		if major != 3 {
			return nil, fmt.Errorf("connector schedules require Payments >= 3.0.0")
		}
		op.Segments = append([]string{"v3"}, op.Segments...)
		return cursorOperation(ctx, client, r, op, body)
	}
	if major == 3 {
		op, body, err = prepareConnectorV3(ctx, client, r, op, body)
	} else {
		op, err = prepareConnectorV1(ctx, client, r, op, major)
		if op.Command == "connectors list" {
			query = nil
		}
	}
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
	}
	path, err := commandapi.Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	data, err := client.Do(ctx, op.Method, path, query, body, nil)
	if err == nil && op.Command == "connectors list" && major < 3 {
		return connectorChoiceIDs(data)
	}
	return data, err
}

func connectorChoiceIDs(data json.RawMessage) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(envelope["data"], &rows); err != nil || rows == nil {
		return nil, fmt.Errorf("connectors response is missing a data array")
	}
	for _, row := range rows {
		if _, exists := row["id"]; !exists {
			row["id"] = row["connectorID"]
		}
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	envelope["data"] = encoded
	return json.Marshal(envelope)
}

func canonicalProvider(ctx context.Context, client *httpclient.Client, requested string) (string, error) {
	configs, err := client.Do(ctx, http.MethodGet, httpclient.Path("v3", "connectors", "configs"), nil, nil, nil)
	if err != nil {
		return "", err
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(configs, &envelope) != nil || envelope.Data == nil {
		return "", fmt.Errorf("connector configs response is missing a data object")
	}
	for provider := range envelope.Data {
		if strings.EqualFold(provider, requested) {
			return provider, nil
		}
	}
	// The historical CLI falls back to the requested provider when configs
	// omits it (for example a development connector). The install/update API
	// remains authoritative and its rejection is returned intact.
	return requested, commandapi.Identifier(requested)
}

func discoverLegacyConnector(ctx context.Context, client *httpclient.Client, provider, id string) (string, string, error) {
	data, err := client.Do(ctx, http.MethodGet, httpclient.Path("connectors"), nil, nil, nil)
	if err != nil {
		return "", "", err
	}
	var envelope struct {
		Data []struct {
			ID       string `json:"connectorID"`
			Provider string `json:"provider"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", "", err
	}
	if provider != "" {
		provider, err = legacyProvider(provider)
		if err != nil {
			return "", "", err
		}
	}
	count := 0
	foundProvider, foundID := "", ""
	for _, connector := range envelope.Data {
		if id != "" && id != connector.ID {
			continue
		}
		if provider != "" && !strings.EqualFold(provider, connector.Provider) {
			continue
		}
		count++
		foundProvider, foundID = connector.Provider, connector.ID
	}
	if count != 1 {
		return "", "", fmt.Errorf("expected one connector, found %d; specify --provider and --connector-id", count)
	}
	return foundProvider, foundID, nil
}

func prepareConnectorV3(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, body json.RawMessage) (commandapi.Operation, json.RawMessage, error) {
	op.Segments = append([]string{"v3"}, op.Segments...)
	switch op.Command {
	case "connectors get-config", "connectors uninstall", "connectors update-config":
		id := r.Flags["connector-id"]
		if err := commandapi.Identifier(id); err != nil {
			return op, nil, fmt.Errorf("--connector-id is required: %w", err)
		}
		op.Segments = []string{"v3", "connectors", id}
		if op.Command != "connectors uninstall" {
			op.Segments = append(op.Segments, "config")
		}
		if op.Command == "connectors update-config" {
			op.Method = http.MethodPatch
		}
	}
	if op.Command != "connectors install" && op.Command != "connectors update-config" {
		return op, body, nil
	}
	provider, err := canonicalProvider(ctx, client, r.Args[0])
	if err != nil {
		return op, nil, err
	}
	body, err = connectorConfig(body, provider)
	if err != nil {
		return op, nil, err
	}
	if op.Command == "connectors install" {
		op.Segments = []string{"v3", "connectors", "install", strings.ToLower(provider)}
	}
	return op, body, nil
}

func connectorConfig(body json.RawMessage, provider string) (json.RawMessage, error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}
	value, err := json.Marshal(provider)
	if err != nil {
		return nil, err
	}
	config["provider"] = value
	return json.Marshal(config)
}

func prepareConnectorV1(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, major int) (commandapi.Operation, error) {
	switch op.Command {
	case "connectors install", "connectors update-config":
		provider, err := legacyProvider(r.Args[0])
		if err != nil {
			return op, err
		}
		op.Segments = []string{"connectors", provider}
		if op.Command == "connectors update-config" {
			id := r.Flags["connector-id"]
			if err := commandapi.Identifier(id); err != nil {
				return op, fmt.Errorf("--connector-id is required")
			}
			op.Segments = append(op.Segments, id, "config")
		}
	case "connectors get-config", "connectors uninstall":
		path, err := legacyConnectorPath(ctx, client, r, op.Command, major)
		if err != nil {
			return op, err
		}
		op.Segments = path
	}
	return op, nil
}

func legacyConnectorPath(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, command string, major int) ([]string, error) {
	provider, id := r.Flags["provider"], r.Flags["connector-id"]
	var err error
	if major != 0 && command == "connectors get-config" && (provider == "" || id == "") {
		provider, id, err = discoverLegacyConnector(ctx, client, provider, id)
		if err != nil {
			return nil, err
		}
	}
	provider, err = legacyProvider(provider)
	if err != nil {
		return nil, err
	}
	path := []string{"connectors", provider}
	if major != 0 {
		if err := commandapi.Identifier(id); err != nil {
			return nil, fmt.Errorf("--connector-id is required")
		}
		path = append(path, id)
	}
	if command == "connectors get-config" {
		path = append(path, "config")
	}
	return path, nil
}

func cursorOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, body json.RawMessage) (json.RawMessage, error) {
	query, err := commandapi.QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
		body = nil
	}
	path, err := commandapi.Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, op.Method, path, query, body, nil)
}
