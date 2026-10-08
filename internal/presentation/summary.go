package presentation

import (
	"fmt"
	"slices"
	"strings"
)

const summaryWidth = 120

// Only nested values have a display budget. The original JSON and top-level
// scalars never pass through this summarizer, including exact numeric tokens.
func (r *renderer) flatScalarOrSummary(value any) string {
	if !nested(value) {
		return r.flat(value)
	}
	r.summarized = true
	return r.summary(value, 0)
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
	if len(object) == 0 {
		return "No fields"
	}
	if depth >= 2 {
		return summaryCount(len(object), "field")
	}
	keys := summaryKeys(object)
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
		return summaryCount(len(object), "field")
	}
	if remaining := len(object) - len(parts); remaining > 0 {
		parts = append(parts, "+"+summaryCount(remaining, "field"))
	}
	return strings.Join(parts, "; ")
}

func (r *renderer) arraySummary(items []any, depth int) string {
	count := summaryCount(len(items), "item")
	if len(items) == 0 || depth >= 1 {
		return count
	}
	parts := []string{count}
	shown := 0
	for _, item := range items[:min(3, len(items))] {
		part := r.summary(item, depth+1)
		if !summaryFits(parts, part) {
			break
		}
		parts = append(parts, part)
		shown++
	}
	if remaining := len(items) - shown; remaining > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", remaining))
	}
	return strings.Join(parts, "; ")
}

func diagnosticText(key string, object map[string]any) bool {
	if !slices.Contains([]string{"message", "description"}, strings.ToLower(key)) {
		return false
	}
	// Named resources and typed conditions already have useful identifiers.
	// Their verbose diagnostic text is available in the full JSON output.
	return object["name"] != nil || object["type"] != nil
}

func summaryFits(parts []string, part string) bool {
	// Reserve space for the omitted-field/item count; never cut numeric tokens.
	return textWidth(strings.Join(append(slices.Clone(parts), part), "; ")) <= summaryWidth-16
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
