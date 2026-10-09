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
