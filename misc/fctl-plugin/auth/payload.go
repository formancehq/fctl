package auth

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

var clientFieldFlags = map[string]string{
	"name": "name", "description": "description", "public": "public", "trusted": "trusted",
	"redirect-uri": "redirectUris", "post-logout-redirect-uri": "postLogoutRedirectUris", "client-scopes": "scopes",
}

func bodyObject(req pluginsdk.ExecuteRequest) (map[string]json.RawMessage, error) {
	if req.Flags["data"] != "" && req.Body == nil {
		return nil, fmt.Errorf("--data must be decoded into Body by the host")
	}
	fields := make(map[string]json.RawMessage)
	if req.Body != nil {
		if err := json.Unmarshal(req.Body, &fields); err != nil || fields == nil {
			return nil, fmt.Errorf("--data must be a JSON object")
		}
	}
	return fields, nil
}

func clientFields(req pluginsdk.ExecuteRequest, create bool) (json.RawMessage, error) {
	fields, err := bodyObject(req)
	if err != nil {
		return nil, err
	}
	if err := applyClientFlags(fields, req); err != nil {
		return nil, err
	}
	if create {
		if err := initializeClientFields(fields, req.Args); err != nil {
			return nil, err
		}
	}
	if err := validateFields(fields, create, false); err != nil {
		return nil, err
	}
	if !create && len(fields) == 0 {
		return nil, fmt.Errorf("provide at least one client option using --data or an explicit update flag")
	}
	return json.Marshal(fields)
}

func applyClientFlags(fields map[string]json.RawMessage, req pluginsdk.ExecuteRequest) error {
	for flag, field := range clientFieldFlags {
		if !req.ChangedFlags[flag] {
			continue
		}
		if _, exists := fields[field]; exists {
			return fmt.Errorf("provide %s using either --data or --%s", field, flag)
		}
		value, err := encodeFlag(flag, req.Flags[flag])
		if err != nil {
			return err
		}
		fields[field] = value
	}
	return nil
}

func initializeClientFields(fields map[string]json.RawMessage, args []string) error {
	if len(args) > 0 {
		if err := setName(fields, args[0]); err != nil {
			return err
		}
	}
	for _, field := range []string{"public", "trusted"} {
		if _, exists := fields[field]; !exists {
			fields[field] = json.RawMessage("false")
		}
	}
	return nil
}

func applyName(fields map[string]json.RawMessage, req pluginsdk.ExecuteRequest, index int) error {
	if req.ChangedFlags["name"] {
		if err := setName(fields, req.Flags["name"]); err != nil {
			return err
		}
	}
	if len(req.Args) > index {
		return setName(fields, req.Args[index])
	}
	return nil
}

func setName(fields map[string]json.RawMessage, name string) error {
	if _, exists := fields["name"]; exists {
		return fmt.Errorf("provide name using only one of the positional argument, --name or --data")
	}
	value, err := json.Marshal(name)
	if err != nil {
		return err
	}
	fields["name"] = value
	return nil
}

func encodeFlag(flag, value string) (json.RawMessage, error) {
	switch flag {
	case "public", "trusted":
		return json.RawMessage(value), nil // NormalizeRequest validated the boolean.
	case "redirect-uri", "post-logout-redirect-uri", "client-scopes":
		values := []string{}
		if value != "" {
			records, err := csv.NewReader(strings.NewReader(value)).ReadAll()
			if err != nil || len(records) != 1 {
				return nil, fmt.Errorf("--%s must contain one valid CSV record", flag)
			}
			values = records[0]
		}
		return json.Marshal(values)
	default:
		return json.Marshal(value)
	}
}

func validateFields(fields map[string]json.RawMessage, requireName, secret bool) error {
	if requireName {
		if _, exists := fields["name"]; !exists {
			return fmt.Errorf("name is required in the positional argument, --name or --data")
		}
	}
	for field, raw := range fields {
		if secret && field != "name" && field != "metadata" {
			return fmt.Errorf("unsupported secret option %q", field)
		}
		if err := validateField(field, raw); err != nil {
			return err
		}
	}
	return nil
}

func validateField(field string, raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%s must not be null", field)
	}
	switch field {
	case "name", "description":
		return validateStringField(field, raw)
	case "public", "trusted":
		var value bool
		if json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%s must be a boolean", field)
		}
		return nil
	case "redirectUris", "postLogoutRedirectUris", "scopes":
		return validateStringArray(field, raw)
	case "metadata":
		return validateMetadata(raw)
	default:
		return fmt.Errorf("unsupported client option %q", field)
	}
}

func validateStringField(field string, raw json.RawMessage) error {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return fmt.Errorf("%s must be a string", field)
	}
	if field == "name" && strings.TrimSpace(value) == "" {
		return fmt.Errorf("name must not be empty")
	}
	return nil
}

func validateStringArray(field string, raw json.RawMessage) error {
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return fmt.Errorf("%s must be an array of strings", field)
	}
	for _, value := range values {
		var text string
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &text) != nil {
			return fmt.Errorf("%s must be an array of strings", field)
		}
	}
	return nil
}

func validateMetadata(raw json.RawMessage) error {
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil || values == nil {
		return fmt.Errorf("metadata must be an object of strings")
	}
	for _, value := range values {
		var text string
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &text) != nil {
			return fmt.Errorf("metadata must be an object of strings")
		}
	}
	return nil
}

func updateClient(ctx context.Context, client *httpclient.Client, path string, req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	patch, err := clientFields(req, false)
	if err != nil {
		return nil, err
	}
	data, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var current struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &current); err != nil || current.Data == nil {
		return nil, fmt.Errorf("read client response must contain a data object")
	}
	// PUT replaces the options. Retain writable fields only, excluding secrets
	// and the server's ID; never replace the name with the positional client ID.
	fields := make(map[string]json.RawMessage)
	for _, field := range clientFieldFlags {
		if value, exists := current.Data[field]; exists && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			fields[field] = value
		}
	}
	if value, exists := current.Data["metadata"]; exists && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		fields["metadata"] = value
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	maps.Copy(fields, changes)
	if err := validateFields(fields, true, false); err != nil {
		return nil, fmt.Errorf("invalid client options after update: %w", err)
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, http.MethodPut, path, nil, body, nil)
}
