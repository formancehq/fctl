package ledger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func flagPairs(value string) (map[string]string, error) {
	if value == "" {
		return map[string]string{}, nil
	}
	var entries []string
	if strings.HasPrefix(strings.TrimSpace(value), "[") {
		if err := json.Unmarshal([]byte(value), &entries); err != nil {
			return nil, fmt.Errorf("expected a JSON string array: %w", err)
		}
	} else {
		reader := csv.NewReader(strings.NewReader(value))
		reader.FieldsPerRecord = -1
		var err error
		entries, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("parse comma-separated pairs: %w", err)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("use a JSON string array for multiline values")
		}
	}
	return pairs(entries)
}

func pairs(entries []string) (map[string]string, error) {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("malformed key=value pair %q", entry)
		}
		result[key] = value
	}
	return result, nil
}

func filterPayload(metadata, address string) (json.RawMessage, error) {
	values, err := flagPairs(metadata)
	if err != nil {
		return nil, err
	}
	// Stable ordering keeps fixtures and request fingerprints reproducible.
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	clauses := make([]map[string]any, 0, len(keys)+1)
	for _, key := range keys {
		clauses = append(clauses, map[string]any{"$match": map[string]string{"metadata[" + key + "]": values[key]}})
	}
	if address != "" {
		clauses = append(clauses, map[string]any{"$match": map[string]string{"account": address}})
	}
	return json.Marshal(map[string]any{"$and": clauses})
}

func sendPayload(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	source := "world"
	args := r.Args
	if len(args) == 4 {
		source, args = args[0], args[1:]
	}
	if strings.TrimSpace(source) == "" || strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[2]) == "" {
		return nil, fmt.Errorf("source, destination and asset must be nonempty")
	}
	amount, err := nonnegativeInteger(args[1], "amount")
	if err != nil {
		return nil, err
	}
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"postings":  []map[string]any{{"source": source, "destination": args[0], "amount": amount, "asset": args[2]}},
		"reference": r.Flags["reference"],
	}
	if len(metadata) > 0 {
		body["metadata"] = metadata
	}
	return json.Marshal(body)
}

func numPayload(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body, err := scriptBody(r.Body)
	if err != nil {
		return nil, err
	}
	script, err := scriptWithVariables(body["script"], r.Flags)
	if err != nil {
		return nil, err
	}
	body["script"] = script
	if err := transactionBodyFlags(body, r); err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

func scriptWithVariables(raw json.RawMessage, flags map[string]string) (json.RawMessage, error) {
	var script map[string]json.RawMessage
	if err := json.Unmarshal(raw, &script); err != nil || script == nil {
		return nil, fmt.Errorf("numscript body must contain a script object")
	}
	var plain string
	if err := json.Unmarshal(script["plain"], &plain); err != nil || strings.TrimSpace(plain) == "" {
		return nil, fmt.Errorf("numscript plain text must be nonempty; the host must read the source into Body")
	}
	vars := map[string]json.RawMessage{}
	if raw, exists := script["vars"]; exists {
		if err := json.Unmarshal(raw, &vars); err != nil || vars == nil {
			return nil, fmt.Errorf("script vars must be a JSON object")
		}
	}
	if err := applyVariables(vars, flags); err != nil {
		return nil, err
	}
	if len(vars) > 0 {
		data, err := json.Marshal(vars)
		if err != nil {
			return nil, err
		}
		script["vars"] = data
	}
	return json.Marshal(script)
}

func applyVariables(vars map[string]json.RawMessage, flags map[string]string) error {
	for _, flag := range []string{"account-var", "portion-var", "amount-var"} {
		entries, err := flagPairs(flags[flag])
		if err != nil {
			return fmt.Errorf("--%s: %w", flag, err)
		}
		for name, value := range entries {
			data, err := variableJSON(flag, value)
			if err != nil {
				return fmt.Errorf("--%s variable %s: %w", flag, name, err)
			}
			vars[name] = data
		}
	}
	return nil
}

func variableJSON(flag, value string) (json.RawMessage, error) {
	if flag != "amount-var" {
		return json.Marshal(value)
	}
	amountValue, asset, ok := strings.Cut(value, "/")
	if !ok || asset == "" {
		return nil, fmt.Errorf("expected name=amount/asset")
	}
	amount, err := nonnegativeInteger(amountValue, "amount variable")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"amount": amount, "asset": asset})
}

func scriptBody(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("numscript Body is required; file/stdin reads belong to the host")
	}
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		script, err := json.Marshal(map[string]string{"plain": plain})
		return map[string]json.RawMessage{"script": script}, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return nil, fmt.Errorf("numscript Body must be a JSON string or object")
	}
	if _, hasPlain := body["plain"]; hasPlain {
		return map[string]json.RawMessage{"script": raw}, nil
	}
	if _, exists := body["postings"]; exists {
		return nil, fmt.Errorf("numscript Body cannot also contain postings")
	}
	return body, nil
}

func transactionBodyFlags(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if err := metadataBodyFlag(body, r); err != nil {
		return err
	}
	if err := referenceBodyFlag(body, r); err != nil {
		return err
	}
	return timestampBodyFlag(body, r)
}

func metadataBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return err
	}
	if len(metadata) > 0 || r.ChangedFlags["metadata"] {
		body["metadata"], err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	} else if raw, exists := body["metadata"]; exists {
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil || values == nil {
			return fmt.Errorf("transaction metadata must be a JSON object")
		}
	}
	return nil
}

func referenceBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if _, exists := body["reference"]; !exists || r.Flags["reference"] != "" || r.ChangedFlags["reference"] {
		data, err := json.Marshal(r.Flags["reference"])
		if err != nil {
			return err
		}
		body["reference"] = data
	}
	var reference string
	if err := json.Unmarshal(body["reference"], &reference); err != nil {
		return fmt.Errorf("transaction reference must be a string")
	}
	return nil
}

func timestampBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if r.Flags["timestamp"] != "" {
		data, err := json.Marshal(r.Flags["timestamp"])
		if err != nil {
			return err
		}
		body["timestamp"] = data
	} else if r.ChangedFlags["timestamp"] {
		delete(body, "timestamp")
	}
	if raw, exists := body["timestamp"]; exists {
		var timestamp string
		if err := json.Unmarshal(raw, &timestamp); err != nil {
			return fmt.Errorf("transaction timestamp must be a string")
		}
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			return fmt.Errorf("parsing transaction timestamp: %w", err)
		}
	}
	return nil
}

func validateSchema(raw json.RawMessage) error {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil || schema == nil {
		return fmt.Errorf("schema Body must be a JSON object; the host must read and convert the source")
	}
	for _, name := range []string{"chart", "queries", "transactions"} {
		value, exists := schema[name]
		if !exists {
			if name == "chart" {
				return fmt.Errorf("schema chart is required")
			}
			continue
		}
		if err := validateSchemaEntries(name, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSchemaEntries(name string, raw json.RawMessage) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || entries == nil {
		return fmt.Errorf("schema %s must be a JSON object", name)
	}
	for key, entry := range entries {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(entry, &object); err != nil || object == nil {
			return fmt.Errorf("schema %s[%s] must be a JSON object", name, key)
		}
	}
	return nil
}
