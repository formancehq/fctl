// Package presentation renders exact JSON values as JSON or readable tables.
package presentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Options controls formatting. The host resolves auto based on its terminal;
// an unresolved auto uses JSON. Width is the table line width, defaulting to 100.
type Options struct {
	Format string
	Color  bool
	Width  int
}

// ValidateFormat accepts the host's automatic mode and explicit output modes.
func ValidateFormat(format string) error {
	if format == "auto" || format == "json" || format == "table" {
		return nil
	}
	return fmt.Errorf("unsupported output format %q: use auto, json or table", format)
}

// Render validates before writing. JSON indentation preserves numeric tokens;
// details summarize nested fields. Lists select scalar columns and explicitly
// report omitted fields. Scalar values wrap rather than truncate.
func Render(out io.Writer, data json.RawMessage, options Options) error {
	if err := ValidateFormat(options.Format); err != nil {
		return err
	}
	if out == nil {
		return fmt.Errorf("output writer is required")
	}
	if !json.Valid(data) {
		return fmt.Errorf("response is not valid JSON")
	}
	if options.Format != "table" {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, data, "", "  "); err != nil {
			return fmt.Errorf("indent response: %w", err)
		}
		return writeOutput(out, strings.TrimRight(formatted.String(), "\r\n")+"\n")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if options.Width <= 0 {
		options.Width = 100
	}
	r := renderer{options: options}
	r.value(value)
	if r.summarized {
		r.wrapped("Use -o json for full nested fields.")
	}
	if r.err != nil {
		return r.err
	}
	return writeOutput(out, r.output.String())
}

func writeOutput(out io.Writer, text string) error {
	n, err := io.WriteString(out, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}

type renderer struct {
	options    Options
	output     strings.Builder
	err        error
	summarized bool
}

func (r *renderer) line(text string) {
	if r.err == nil {
		_, r.err = r.output.WriteString(text + "\n")
	}
}

func (r *renderer) title(text string) {
	for _, line := range wrap(text, r.options.Width) {
		r.line(r.style(line, "1;36"))
	}
}

func (r *renderer) value(value any) {
	switch typed := value.(type) {
	case map[string]any:
		r.object(typed)
	case []any:
		r.list(typed)
	default:
		r.title("Result")
		r.scalar(typed)
	}
}

func (r *renderer) scalar(value any) {
	text := r.flat(value)
	if flag, ok := value.(bool); ok {
		if flag {
			text = "Success (true)."
		} else {
			text = "Not successful (false)."
		}
	}
	for _, line := range wrap(text, r.options.Width) {
		r.line(r.status(line))
	}
}

func (r *renderer) object(object map[string]any) {
	remaining := maps.Clone(object)
	if data, exists := remaining["data"]; exists {
		delete(remaining, "data")
		r.value(data)
	} else if cursor, ok := remaining["cursor"].(map[string]any); ok {
		delete(remaining, "cursor")
		r.value(cursor)
	}
	if len(remaining) != 0 {
		r.title("Details")
		r.details(remaining)
	} else if len(object) == 0 {
		r.title("Details")
		r.wrapped("No fields.")
	}
}

func (r *renderer) details(object map[string]any) {
	rows := make([][]string, 0, len(object))
	for _, key := range orderedKeys(object) {
		value := object[key]
		text := r.flatScalarOrSummary(value)
		rows = append(rows, []string{fieldLabel(key), text})
	}
	r.table([]string{"Field", "Value"}, rows)
}

func (r *renderer) list(items []any) {
	r.title("Results")
	if len(items) == 0 {
		r.wrapped("No results.")
		return
	}
	keys, omitted, objects := listColumns(items, r.options.Width)
	if !objects {
		r.records(items)
		return
	}
	if len(keys) == 0 {
		r.recordNumbers(len(items))
		r.omittedFields(omitted)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			r.err = fmt.Errorf("list item is not an object")
			return
		}
		row := make([]string, len(keys))
		for i, key := range keys {
			if value, exists := object[key]; exists {
				row[i] = r.flat(value)
			}
		}
		rows = append(rows, row)
	}
	headers := make([]string, len(keys))
	for i, key := range keys {
		headers[i] = fieldLabel(key)
	}
	r.table(headers, rows)
	r.omittedFields(omitted)
}

func (r *renderer) recordNumbers(count int) {
	rows := make([][]string, count)
	for i := range count {
		rows[i] = []string{strconv.Itoa(i + 1)}
	}
	r.table([]string{"Record"}, rows)
}

func (r *renderer) omittedFields(count int) {
	if count == 0 {
		return
	}
	unit := "fields"
	if count == 1 {
		unit = "field"
	}
	r.wrapped(fmt.Sprintf("%d additional %s omitted.", count, unit))
	r.wrapped("Use -o json for all fields.")
}

func (r *renderer) records(items []any) {
	for i, item := range items {
		r.title("Record " + strconv.Itoa(i+1))
		if object, ok := item.(map[string]any); ok && len(object) != 0 {
			r.details(object)
		} else {
			r.scalar(item)
		}
	}
}

func listFields(items []any) (map[string]any, bool) {
	fields := make(map[string]any)
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		for key, value := range object {
			if !nested(fields[key]) {
				fields[key] = value
			}
		}
	}
	return fields, true
}

func nested(value any) bool {
	switch value.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func orderedKeys(object map[string]any) []string {
	keys := slices.Sorted(maps.Keys(object))
	preferred := []string{"id", "name", "state", "status", "region", "regionid", "version", "createdat", "asset", "account", "amount", "balance"}
	slices.SortStableFunc(keys, func(a, b string) int {
		return fieldRank(a, object[a], preferred) - fieldRank(b, object[b], preferred)
	})
	return keys
}

func fieldRank(key string, value any, preferred []string) int {
	switch value.(type) {
	case map[string]any, []any:
		return len(preferred) + 1
	default:
		if i := slices.Index(preferred, strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))); i >= 0 {
			return i
		}
		return len(preferred)
	}
}

func (r *renderer) wrapped(text string) {
	for _, line := range wrap(text, r.options.Width) {
		r.line(line)
	}
}

func (r *renderer) flat(value any) string {
	if text, ok := value.(string); ok {
		return escape(text)
	}
	data, err := json.Marshal(value)
	if err != nil {
		r.err = fmt.Errorf("render field: %w", err)
		return ""
	}
	return escape(string(data))
}

func (r *renderer) style(text, code string) string {
	if !r.options.Color || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (r *renderer) status(text string) string {
	switch strings.ToLower(text) {
	case "true", "success", "active", "ready", "running", "approved", "enabled", "succeeded", "success (true).":
		return r.style(text, "32")
	case "false", "error", "failed", "denied", "degraded", "unreachable", "not successful (false).":
		return r.style(text, "31")
	case "pending", "starting", "stopped", "progressing", "unknown", "disabled", "deleted", "deleting", "restoring", "upgrading":
		return r.style(text, "33")
	default:
		return text
	}
}
