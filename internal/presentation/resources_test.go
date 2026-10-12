package presentation_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/internal/presentation"
)

func TestResourceIdentityTablesPreserveJSON(t *testing.T) {
	t.Parallel()
	body := json.RawMessage(`{"cursor":{"hasMore":false,"data":[{"metadata":{"name":"ingestion","namespace":"sandbox"},"spec":{"startSequence":9007199254740993},"status":{"phase":"Running"}}]}}`)
	var table, raw bytes.Buffer
	if err := presentation.Render(&table, body, presentation.Options{Format: "table"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ingestion", "sandbox", "Running", "Name", "Phase"} {
		if !strings.Contains(table.String(), want) {
			t.Errorf("missing %s: %s", want, table.String())
		}
	}
	if err := presentation.Render(&raw, body, presentation.Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if compact.String() != string(body) {
		t.Fatal("table projection changed raw JSON")
	}
}

func TestPluginListShowsModuleBeforeOtherColumns(t *testing.T) {
	t.Parallel()
	body := `[{"profile":"cloud","organization":"jdxmvkvwlyiy","stack":"bwxm","endpoint":"https://app.formance.cloud/api","service":"ledger","version":"3.0.0-beta.10","revision":1,"platform":"darwin/arm64","source":"local","sha256":"abc123"}]`
	for _, width := range []int{40, 80, 160, 320} {
		out := render(t, body, presentation.Options{Format: "table", Width: width})
		assertContains(t, out, "Module", "ledger")
		header := strings.Split(out, "\n")[2]
		if !strings.HasPrefix(header, "| Module") {
			t.Fatalf("module is not the first column at width %d: %s", width, header)
		}
		assertLineWidths(t, out, width)
	}
	out := render(t, body, presentation.Options{Format: "json"})
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(out)); err != nil {
		t.Fatal(err)
	}
	if compact.String() != body {
		t.Fatal("plugin table projection changed JSON output")
	}
}
