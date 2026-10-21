package plugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func pointerParts(pointer string) ([]string, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("body input requires a JSON pointer")
	}
	parts := strings.Split(pointer[1:], "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, nil
}

func bodyValue(body json.RawMessage, pointer string) (any, bool) {
	parts, err := pointerParts(pointer)
	if err != nil {
		return nil, false
	}
	value, err := decodeJSON(body)
	if err != nil {
		return nil, false
	}
	for _, part := range parts {
		switch container := value.(type) {
		case map[string]any:
			var exists bool
			value, exists = container[part]
			if !exists {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(container) {
				return nil, false
			}
			value = container[index]
		default:
			return nil, false
		}
	}
	return value, true
}

func setBodyValue(body json.RawMessage, pointer string, value json.RawMessage) (json.RawMessage, error) {
	parts, err := pointerParts(pointer)
	if err != nil {
		return nil, err
	}
	root, err := decodeJSON(body)
	if err != nil {
		return nil, err
	}
	root, err = setJSONValue(root, parts, value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(root)
}

func setJSONValue(node any, parts []string, value json.RawMessage) (any, error) {
	if len(parts) == 0 {
		return value, nil
	}
	key := parts[0]
	if node == nil {
		if _, err := strconv.Atoi(key); err == nil {
			node = []any{}
		} else {
			node = map[string]any{}
		}
	}
	switch container := node.(type) {
	case map[string]any:
		child, err := setJSONValue(container[key], parts[1:], value)
		if err != nil {
			return nil, err
		}
		container[key] = child
		return container, nil
	case []any:
		return setArrayValue(container, key, parts[1:], value)
	default:
		return nil, fmt.Errorf("form input conflicts with an existing JSON value")
	}
}

func setArrayValue(container []any, key string, parts []string, value json.RawMessage) (any, error) {
	index, err := strconv.Atoi(key)
	if err != nil || index < 0 || index > 1000 {
		return nil, fmt.Errorf("invalid form array index")
	}
	for len(container) <= index {
		container = append(container, nil)
	}
	child, err := setJSONValue(container[index], parts, value)
	if err != nil {
		return nil, err
	}
	container[index] = child
	return container, nil
}
