package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
)

// RenderFromSchema renders structured JSON data using the display schema
// declared in the plugin's command manifest.
func RenderFromSchema(out io.Writer, jsonData string, schema *pluginpb.DisplaySchema, outputFormat string) error {
	switch outputFormat {
	case "json":
		return renderJSON(out, jsonData)
	case "yaml":
		return renderYAML(out, jsonData)
	default:
		return renderPlain(out, jsonData, schema)
	}
}

func renderJSON(out io.Writer, jsonData string) error {
	var raw any
	if err := json.Unmarshal([]byte(jsonData), &raw); err != nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}
	formatted, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}
	_, err = fmt.Fprintln(out, string(formatted))
	return err
}

func renderYAML(out io.Writer, jsonData string) error {
	var raw any
	if err := json.Unmarshal([]byte(jsonData), &raw); err != nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}
	yamlBytes, err := yaml.Marshal(raw)
	if err != nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}
	_, err = out.Write(yamlBytes)
	return err
}

func renderPlain(out io.Writer, jsonData string, schema *pluginpb.DisplaySchema) error {
	if schema == nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}

	if len(schema.Columns) > 0 {
		return renderTable(out, jsonData, schema.Columns)
	}

	if len(schema.Sections) > 0 {
		return renderSections(out, jsonData, schema.Sections)
	}

	_, err := fmt.Fprintln(out, jsonData)
	return err
}

func renderTable(out io.Writer, jsonData string, columns []*pluginpb.ColumnSpec) error {
	var items []map[string]any

	// Try array first, then single object
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		var single map[string]any
		if err := json.Unmarshal([]byte(jsonData), &single); err != nil {
			_, err := fmt.Fprintln(out, jsonData)
			return err
		}
		items = []map[string]any{single}
	}

	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "No results.")
		return err
	}

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}

	tableData := pterm.TableData{headers}
	for _, item := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			val := extractJSONPath(item, col.JsonPath)
			row[i] = formatValue(val, col.Format)
		}
		tableData = append(tableData, row)
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}

func renderSections(out io.Writer, jsonData string, sections []*pluginpb.SectionSpec) error {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		_, err := fmt.Fprintln(out, jsonData)
		return err
	}

	for _, section := range sections {
		if section.Title != "" {
			fmt.Fprintf(out, "%s\n", section.Title)
			fmt.Fprintf(out, "%s\n", strings.Repeat("─", len(section.Title)))
		}
		for _, field := range section.Fields {
			val := extractJSONPath(data, field.JsonPath)
			fmt.Fprintf(out, "  %-20s %s\n", field.Label+":", formatValue(val, field.Format))
		}
		fmt.Fprintln(out)
	}

	return nil
}

// extractJSONPath extracts a value from a map using a dot-separated path.
// Supports simple paths like "id", "metadata.key", "postings.0.amount".
func extractJSONPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		case []any:
			idx := 0
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx < len(v) {
				current = v[idx]
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

func formatValue(val any, format string) string {
	if val == nil {
		return "-"
	}

	switch format {
	case "timestamp":
		if s, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.Format(time.RFC3339)
			}
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t.Format(time.RFC3339)
			}
		}
	case "number":
		if f, ok := val.(float64); ok {
			if f == float64(int64(f)) {
				return fmt.Sprintf("%d", int64(f))
			}
			return fmt.Sprintf("%.2f", f)
		}
	}

	return fmt.Sprintf("%v", val)
}
