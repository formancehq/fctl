package command

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

func ObjectBody(body json.RawMessage, required ...string) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if len(body) == 0 || json.Unmarshal(body, &object) != nil || object == nil {
		return nil, fmt.Errorf("a JSON object payload is required (--data)")
	}
	for _, field := range required {
		v := strings.TrimSpace(string(object[field]))
		if v == "" || v == "null" || v == `""` {
			return nil, fmt.Errorf("payload field %q is required", field)
		}
	}
	return body, nil
}

// list accepts the SDK's string flag representation as JSON or CSV.
func List(value string) ([]string, error) {
	if value == "" {
		return []string{}, nil
	}
	var values []string
	if strings.HasPrefix(strings.TrimSpace(value), "[") {
		if err := json.Unmarshal([]byte(value), &values); err != nil {
			return nil, fmt.Errorf("expected a JSON string array: %w", err)
		}
	} else {
		var err error
		values, err = csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return nil, fmt.Errorf("expected comma-separated values: %w", err)
		}
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("list entries cannot be empty")
		}
	}
	return values, nil
}

func Pairs(value string) (map[string]string, error) {
	if strings.HasPrefix(strings.TrimSpace(value), "{") {
		var out map[string]string
		if err := json.Unmarshal([]byte(value), &out); err != nil || out == nil {
			return nil, fmt.Errorf("expected a JSON object with string values")
		}
		return out, nil
	}
	values, err := List(value)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, v := range values {
		key, val, ok := strings.Cut(v, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected key=value, got %q", v)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		out[key] = val
	}
	return out, nil
}
