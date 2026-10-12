package presentation

import (
	"fmt"
	"slices"
	"strings"
)

const summaryWidth = 120

// Only nested values have a display budget. The original JSON and top-level
// scalars never pass through this summarizer, including exact numeric tokens.
func (r *renderer) flatScalarOrSummary(key string, value any) string {
	if slices.Contains([]string{"schema", "mirrorSource", "mirrorSyncProgress"}, key) && !hasDetails(value) {
		return "—"
	}
	if !nested(value) {
		return r.flat(value)
	}
	return r.summary(value, 0)
}

// Empty containers, nulls and empty strings carry no configuration details.
// Numeric values and booleans remain meaningful, including zero and false.
func hasDetails(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return typed != ""
	case map[string]any:
		for _, child := range typed {
			if hasDetails(child) {
				return true
			}
		}
		return false
	case []any:
		// A nonempty array retains its size and positions, even for null items.
		return len(typed) != 0
	default:
		return true
	}
}

func (r *renderer) summary(value any, depth int) string {
	switch typed := value.(type) {
	case map[string]any:
		return r.objectSummary(typed, depth)
	case []any:
		return r.arraySummary(typed, depth)
	case string:
		text := escape(typed)
		if chars := []rune(text); len(chars) > 40 {
			return string(chars[:40]) + "..."
		}
		return text
	default:
		return r.flat(value)
	}
}

func (r *renderer) objectSummary(object map[string]any, depth int) string {
	keys := summaryKeys(object)
	keys = slices.DeleteFunc(keys, func(key string) bool { return !hasDetails(object[key]) })
	if len(keys) == 0 {
		return "—"
	}
	if depth >= 2 {
		return summaryCount(len(keys), "field")
	}
	parts := make([]string, 0, 3)
	for _, key := range keys {
		if diagnosticText(key, object) {
			continue
		}
		part := escape(key) + ": " + r.summary(object[key], depth+1)
		if !summaryFits(parts, part) {
			continue
		}
		parts = append(parts, part)
		if len(parts) == 3 {
			break
		}
	}
	if len(parts) == 0 {
		return summaryCount(len(keys), "field")
	}
	return strings.Join(parts, "; ")
}

func (r *renderer) arraySummary(items []any, depth int) string {
	if len(items) == 0 {
		return "—"
	}
	count := summaryCount(len(items), "item")
	if depth >= 1 {
		return count
	}
	parts := []string{count}
	for _, item := range items[:min(3, len(items))] {
		part := r.summary(item, depth+1)
		if !summaryFits(parts, part) {
			break
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

func diagnosticText(key string, object map[string]any) bool {
	if !slices.Contains([]string{"message", "description"}, strings.ToLower(key)) {
		return false
	}
	// Named resources and typed conditions already have useful identifiers.
	// Their verbose diagnostic text is available in the full JSON output.
	return hasDetails(object["name"]) || hasDetails(object["type"])
}

func summaryFits(parts []string, part string) bool {
	// Keep numeric tokens intact when selecting parts for the summary.
	return textWidth(strings.Join(append(slices.Clone(parts), part), "; ")) <= summaryWidth
}

func summaryKeys(object map[string]any) []string {
	keys := orderedKeys(object)
	preferred := []string{"name", "type", "status", "state", "version", "id", "info"}
	slices.SortStableFunc(keys, func(a, b string) int {
		return summaryRank(a, preferred) - summaryRank(b, preferred)
	})
	return keys
}

func summaryRank(key string, preferred []string) int {
	if index := slices.Index(preferred, strings.ToLower(key)); index >= 0 {
		return index
	}
	return len(preferred)
}

func summaryCount(count int, noun string) string {
	if count != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", count, noun)
}
