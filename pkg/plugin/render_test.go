package plugin

import (
	"bytes"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
)

func TestRenderTable(t *testing.T) {
	schema := &pluginpb.DisplaySchema{
		Columns: []*pluginpb.ColumnSpec{
			{Header: "ID", JsonPath: "id", Format: "number"},
			{Header: "Reference", JsonPath: "reference"},
			{Header: "Timestamp", JsonPath: "timestamp", Format: "timestamp"},
		},
	}

	jsonData := `[
		{"id": 42, "reference": "ref-001", "timestamp": "2026-05-14T10:00:00Z"},
		{"id": 41, "reference": null, "timestamp": "2026-05-14T09:45:00Z"}
	]`

	var buf bytes.Buffer
	if err := RenderFromSchema(&buf, jsonData, schema, "plain"); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Fatal("expected header ID in output")
	}
	if !strings.Contains(out, "42") {
		t.Fatal("expected value 42 in output")
	}
	if !strings.Contains(out, "ref-001") {
		t.Fatal("expected ref-001 in output")
	}
}

func TestRenderSections(t *testing.T) {
	schema := &pluginpb.DisplaySchema{
		Sections: []*pluginpb.SectionSpec{
			{
				Title: "Transaction",
				Fields: []*pluginpb.FieldSpec{
					{Label: "ID", JsonPath: "id", Format: "number"},
					{Label: "Reference", JsonPath: "reference"},
					{Label: "Status", JsonPath: "status"},
				},
			},
		},
	}

	jsonData := `{"id": 42, "reference": "ref-001", "status": "committed"}`

	var buf bytes.Buffer
	if err := RenderFromSchema(&buf, jsonData, schema, "plain"); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Transaction") {
		t.Fatal("expected section title")
	}
	if !strings.Contains(out, "42") {
		t.Fatal("expected ID value")
	}
	if !strings.Contains(out, "committed") {
		t.Fatal("expected status value")
	}
}

func TestRenderJSON(t *testing.T) {
	schema := &pluginpb.DisplaySchema{
		Columns: []*pluginpb.ColumnSpec{
			{Header: "ID", JsonPath: "id"},
		},
	}

	jsonData := `[{"id": 1}]`

	var buf bytes.Buffer
	if err := RenderFromSchema(&buf, jsonData, schema, "json"); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, `"id"`) {
		t.Fatal("expected json output")
	}
}

func TestRenderYAML(t *testing.T) {
	schema := &pluginpb.DisplaySchema{
		Columns: []*pluginpb.ColumnSpec{
			{Header: "ID", JsonPath: "id"},
		},
	}

	jsonData := `[{"id": 1, "name": "test"}]`

	var buf bytes.Buffer
	if err := RenderFromSchema(&buf, jsonData, schema, "yaml"); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "name: test") {
		t.Fatalf("expected yaml output, got: %s", out)
	}
}

func TestExtractJSONPathNested(t *testing.T) {
	data := map[string]any{
		"metadata": map[string]any{
			"key": "value",
		},
	}

	val := extractJSONPath(data, "metadata.key")
	if val != "value" {
		t.Fatalf("expected 'value', got %v", val)
	}
}

func TestExtractJSONPathMissing(t *testing.T) {
	data := map[string]any{"id": 1}
	val := extractJSONPath(data, "missing.path")
	if val != nil {
		t.Fatalf("expected nil, got %v", val)
	}
}
