package presentation

import "strings"

func listColumns(items []any, width int) ([]string, int, bool) {
	fields, objects := listFields(items)
	if !objects {
		return nil, 0, false
	}
	var selected []string
	used := 1
	for _, key := range orderedKeys(fields) {
		if nested(fields[key]) || len(selected) == 6 {
			continue
		}
		cost := listColumnWidth(items, key) + 3
		if len(selected) != 0 && used+cost > width {
			continue
		}
		selected = append(selected, key)
		used += cost
	}
	return selected, len(fields) - len(selected), true
}

func listColumnWidth(items []any, key string) int {
	width := max(2, textWidth(fieldLabel(key)))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		// Width estimates only decide which columns fit. Render still displays
		// the complete value, wrapping longer cells without ellipses.
		if value, exists := object[key]; exists {
			width = max(width, min(16, scalarWidth(value)))
		}
	}
	return width
}

func scalarWidth(value any) int {
	if text, ok := value.(string); ok {
		return textWidth(escape(text))
	}
	// The regular field renderer preserves json.Number's original token.
	r := renderer{}
	return textWidth(strings.TrimSpace(r.flat(value)))
}
