package presentation_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/formancehq/fctl/v4/internal/presentation"
)

func render(t *testing.T, data string, options presentation.Options) string {
	t.Helper()
	var out bytes.Buffer
	if err := presentation.Render(&out, json.RawMessage(data), options); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func assertContains(t *testing.T, text string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(text, value) {
			t.Errorf("output missing %q:\n%s", value, text)
		}
	}
}

func TestJSONPreservesNumericTokensAndIndentation(t *testing.T) {
	data := `{"amount":9007199254740993123456789,"negative":-9007199254740993,"decimal":1.234567890123456789,"exponent":1e+100,"data":[true,null]}`
	for _, format := range []string{"json", "auto"} {
		t.Run(format, func(t *testing.T) {
			out := render(t, data, presentation.Options{Format: format, Color: true})
			assertContains(t, out, "9007199254740993123456789", "-9007199254740993", "1.234567890123456789", "1e+100", "\n  ", "\n")
			if !json.Valid([]byte(out)) || strings.Contains(out, "\x1b") {
				t.Fatal("JSON output is invalid or styled")
			}
		})
	}
}

func TestListSelectsScalarFieldsAndPreservesPagination(t *testing.T) {
	data := `{"cursor":{"data":[{"name":"alpha","id":"a","state":"ready","amount":9007199254740993,"payload":{"postings":[{"amount":1234567890123456789}]}},{"id":"b","region":"eu","extra":"only-second-row"}],"hasMore":true,"next":"cursor-next","previous":null,"pageSize":2},"requestId":"request-id"}`
	out := render(t, data, presentation.Options{Format: "table", Width: 250})
	assertContains(t, out, "Results", "ID", "Name", "State", "Region", "alpha", "only-second-row", "9007199254740993", "1 additional field omitted.", "Use -o json for all fields", "Has More", "true", "cursor-next", "Previous", "null", "Page Size", "request-id")
	if strings.Contains(out, "\x1b") || strings.Contains(out, "…") {
		t.Fatal("plain output contains styling or truncated fields")
	}
	header := strings.Split(out, "\n")[2]
	if strings.Index(header, "ID") > strings.Index(header, "Name") || strings.Index(header, "Name") > strings.Index(header, "State") {
		t.Fatalf("preferred fields are not ordered before nested values: %s", header)
	}
}

func TestDetailsSummarizeNestedPayloadAndMetadata(t *testing.T) {
	data := `{"data":{"id":"transaction","payload":{"script":"send [USD/2 9007199254740993]"},"metadata":{"nested":[false,null,{"amount":9007199254740995}]}},"status":"ok","custom":{"request":"trace-id"}}`
	out := render(t, data, presentation.Options{Format: "table", Width: 180})
	assertContains(t, out, "Details", "transaction", "Payload", "9007199254740993", "nested: 3 items", "Status", "ok", "request: trace-id", "Use -o json for full nested fields")
}

func TestHumanReadableFieldLabels(t *testing.T) {
	data := `{"client_id":"client","createdAt":"today","HTTPStatus":200,"hasMore":true,"stackURL":"https://example.test","organizationIDs":["org"]}`
	out := render(t, data, presentation.Options{Format: "table", Width: 120})
	assertContains(t, out, "Client ID", "Created At", "HTTP Status", "Has More", "Stack URL", "Organization IDs")
}

func TestLargeClaimsSummarizedOnlyInTables(t *testing.T) {
	organizations := make([]map[string]any, 40)
	for i := range organizations {
		organizations[i] = map[string]any{"id": fmt.Sprintf("org-%02d", i), "stack": map[string]any{"amount": json.Number("9007199254740993"), "permission": "stack:Read"}}
	}
	data, err := json.Marshal(map[string]any{"claims": map[string]any{"organizations": organizations}})
	if err != nil {
		t.Fatal(err)
	}
	out := render(t, string(data), presentation.Options{Format: "table", Width: 80})
	assertContains(t, out, "Claims", "40 items", "Use -o json for full nested fields")
	assertLineWidths(t, out, 80)
	if strings.Count(out, "\n") > 12 {
		t.Fatalf("claims expanded: %s", out)
	}
	jsonOutput := render(t, string(data), presentation.Options{Format: "json", Color: true})
	for i := range organizations {
		assertContains(t, jsonOutput, fmt.Sprintf("org-%02d", i))
	}
	if strings.Count(jsonOutput, "9007199254740993") != 40 || strings.Contains(jsonOutput, "\x1b") {
		t.Fatal("JSON changed nested numbers or added styling")
	}
}

func TestEmptyListsSuccessAndMixedResults(t *testing.T) {
	for _, tc := range []struct{ data, want string }{
		{`[]`, "No results."},
		{`{"data":[],"hasMore":false,"next":"last-cursor"}`, "last-cursor"},
		{`true`, "Success (true)."},
		{`false`, "Not successful (false)."},
		{`{"success":true,"message":"Client deleted"}`, "Client deleted"},
		{`[{"id":"one"},9007199254740993,"literal",null,[]]`, "9007199254740993"},
		{`{}`, "No fields."},
	} {
		t.Run(tc.data, func(t *testing.T) {
			out := render(t, tc.data, presentation.Options{Format: "table"})
			assertContains(t, out, tc.want)
		})
	}
}

func TestControlCharactersCannotInjectTerminalOutput(t *testing.T) {
	data := `{"name":"a\nb\rc\td\u001b[31mINJECT\u0000","\u001btitle":"\u202eevil","nested":{"raw":"\u001b[2J"}}`
	out := render(t, data, presentation.Options{Format: "table", Width: 200})
	assertContains(t, out, `a\nb\rc\td\u001b[31mINJECT\u0000`, `\u001btitle`, `\u202eevil`, `\u001b[2J`)
	for _, char := range out {
		if char != '\n' && (unicode.IsControl(char) || unicode.Is(unicode.Cf, char)) {
			t.Fatalf("terminal control escaped sanitization: %U", char)
		}
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestWrappingAccountsForWideCharactersAndANSIStyles(t *testing.T) {
	data := `[{"id":"item","name":"界界界界界界","status":"ready","payload":{"amount":9007199254740993}}]`
	for _, width := range []int{1, 8, 20, 32, 60} {
		for _, color := range []bool{false, true} {
			t.Run(fmt.Sprintf("width=%d/color=%t", width, color), func(t *testing.T) {
				out := render(t, data, presentation.Options{Format: "table", Color: color, Width: width})
				plain := ansiPattern.ReplaceAllString(out, "")
				assertLineWidths(t, plain, width)
				if strings.Contains(out, "\x1b") != color {
					t.Fatal("color option not respected")
				}
			})
		}
	}
}

func assertLineWidths(t *testing.T, text string, width int) {
	t.Helper()
	for line := range strings.SplitSeq(text, "\n") {
		if got := visibleWidth(line); got > width {
			t.Fatalf("line width=%d exceeds %d: %q", got, width, line)
		}
	}
}

func visibleWidth(text string) int {
	width := 0
	for _, char := range text {
		if char == '界' {
			width += 2
		} else {
			width++
		}
	}
	return width
}

func TestNarrowRecordsKeepEveryFieldWithoutEllipsis(t *testing.T) {
	data := `[{"id":"one","name":"long-name","region":"eu","extra":"payload-value"}]`
	out := render(t, data, presentation.Options{Format: "table", Width: 19})
	assertContains(t, out, "ID", "Name", "Use -o json")
	if strings.Contains(out, "...") || strings.Contains(out, "…") {
		t.Fatal("narrow rendering truncated values")
	}
}

func TestCloudStackListAtEightyColumnsStaysCompact(t *testing.T) {
	data := `{"data":[{"id":"stack-a","name":"alpha","status":"ready","regionID":"eu","version":"v3","amount":9007199254740993,"configuration":{"services":{"ledger":{"storage":"nested-only"}}},"metadata":{"team":"ops"},"replicas":3},{"id":"stack-b","name":"beta","status":"pending","regionID":"us","version":"v4","amount":9007199254740995,"configuration":{"services":{}},"metadata":{},"replicas":2}],"hasMore":true,"next":"next-page"}`
	out := render(t, data, presentation.Options{Format: "table", Width: 80})
	assertContains(t, out, "stack-a", "stack-b", "alpha", "beta", "ready", "pending", "Region ID", "9007199254740993", "9007199254740995", "3 additional fields omitted.", "Use -o json for all fields", "Has More", "next-page")
	assertLineWidths(t, out, 80)
	if strings.Contains(out, "Record 1") || strings.Contains(out, "nested-only") || strings.Count(out, "\n") > 22 {
		t.Fatalf("list expanded into full records or nested configuration:\n%s", out)
	}
	header := strings.Split(out, "\n")[2]
	if columns := strings.Count(header, "|") - 1; columns > 6 {
		t.Fatalf("list has %d columns", columns)
	}
	jsonOutput := render(t, data, presentation.Options{Format: "json"})
	assertContains(t, jsonOutput, "nested-only", "replicas", "9007199254740993")
}

func TestCloudStackDetailsSummarizeComplexFieldsAtEightyColumns(t *testing.T) {
	conditions := make([]map[string]any, 20)
	for i := range conditions {
		conditions[i] = map[string]any{"type": "Ready", "status": "True", "message": strings.Repeat("condition-blob-", 40), "observedGeneration": json.Number("900719925474099312345")}
	}
	stack := map[string]any{
		"id": "stack-a", "name": "production", "status": "ready", "version": "v4.0",
		"amount":       json.Number("900719925474099312345"),
		"modules":      []any{map[string]any{"name": "ledger", "status": "ready"}, map[string]any{"name": "auth", "status": "ready"}, map[string]any{"name": "payments", "status": "pending"}},
		"cluster":      map[string]any{"name": "eu-primary", "status": "ready", "conditions": conditions},
		"conditions":   conditions,
		"region":       map[string]any{"id": "eu-west", "name": "Europe", "info": map[string]any{"provider": "aws", "location": "Paris"}, "capabilities": []any{"ledger", "auth", "payments", "search"}},
		"capabilities": []any{"ledger", "auth", "payments", "search", "reconciliation", "wallets"},
	}
	data, err := json.Marshal(map[string]any{"data": stack})
	if err != nil {
		t.Fatal(err)
	}
	for _, color := range []bool{false, true} {
		out := render(t, string(data), presentation.Options{Format: "table", Width: 80, Color: color})
		plain := ansiPattern.ReplaceAllString(out, "")
		assertContains(t, plain, "production", "900719925474099312345", "ledger", "auth", "payments", "pending", "eu-primary", "Europe", "20 items", "6 items", "+3 more", "Use -o json for full nested fields")
		assertLineWidths(t, plain, 80)
		if lines := strings.Count(plain, "\n"); lines >= 50 || strings.Contains(plain, "condition-blob") || strings.Contains(plain, `{"`) {
			t.Fatalf("details expanded raw nested data (%d lines):\n%s", lines, plain)
		}
	}
	jsonOutput := render(t, string(data), presentation.Options{Format: "json", Color: true})
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(jsonOutput)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(compact.Bytes(), data) {
		t.Fatal("machine output changed nested fields or numeric tokens")
	}
}

func TestDetailScalarsWrapCompletelyWhileNestedStringsAreBounded(t *testing.T) {
	text := strings.Repeat("complete-scalar", 30)
	data, err := json.Marshal(map[string]any{"description": text, "nested": map[string]any{"name": strings.Repeat("long-nested-name", 50)}})
	if err != nil {
		t.Fatal(err)
	}
	out := render(t, string(data), presentation.Options{Format: "table", Width: 80})
	assertContains(t, detailValues(t, out), text)
	assertContains(t, out, "...", "Use -o json for full nested fields")
	assertLineWidths(t, out, 80)
}

// Reassemble the Value column across wrapped lines without table padding.
func detailValues(t *testing.T, out string) string {
	t.Helper()
	var reconstructed strings.Builder
	for line := range strings.SplitSeq(out, "\n") {
		cells := strings.Split(line, "|")
		if len(cells) == 4 {
			if _, err := reconstructed.WriteString(strings.TrimSpace(cells[2])); err != nil {
				t.Fatal(err)
			}
		}
	}
	return reconstructed.String()
}

func TestNestedSummaryHandlesEmptyDeepAndNumericValues(t *testing.T) {
	data := `{"emptyObject":{},"emptyArray":[],"numeric":{"amount":900719925474099312345,"negative":-9007199254740993,"enabled":false},"deep":{"info":{"info":{"raw":"not-displayed"}}}}`
	out := render(t, data, presentation.Options{Format: "table", Width: 80})
	assertContains(t, out, "No fields", "0 items", "900719925474099312345", "false", "1 field", "Use -o json for full nested fields")
	assertLineWidths(t, out, 80)
	assertContains(t, detailValues(t, out), "amount: 900719925474099312345", "negative: -9007199254740993")
	if strings.Contains(out, "not-displayed") || strings.Count(out, "Use -o json") != 1 {
		t.Fatal("deep field expanded or nested footer repeated")
	}
	plain := render(t, `{"id":"simple","amount":900719925474099312345}`, presentation.Options{Format: "table"})
	if strings.Contains(plain, "Use -o json") {
		t.Fatal("scalar-only details reported summarized fields")
	}
}

func TestNestedOnlyAndHeterogeneousFieldsReportOmissions(t *testing.T) {
	for _, data := range []string{
		`[{"payload":{"secret":"nested-content"}},{"payload":[]}]`,
		`[{"id":"one","payload":7},{"id":"two","payload":{"secret":"nested-content"}}]`,
	} {
		out := render(t, data, presentation.Options{Format: "table", Width: 80})
		assertContains(t, out, "1 additional field omitted.", "Use -o json for all fields")
		if strings.Contains(out, "nested-content") {
			t.Fatal("nested payload expanded in list summary")
		}
	}
}

func TestStatusColorsAndPlainASCIIBorders(t *testing.T) {
	data := `[{"id":"a","status":"ready"},{"id":"b","status":"failed"},{"id":"c","status":"pending"}]`
	styled := render(t, data, presentation.Options{Format: "table", Color: true})
	assertContains(t, styled, "\x1b[1;36m", "\x1b[32mready\x1b[0m", "\x1b[31mfailed\x1b[0m", "\x1b[33mpending\x1b[0m")
	plain := render(t, data, presentation.Options{Format: "table"})
	if styledPlain := ansiPattern.ReplaceAllString(styled, ""); styledPlain != plain {
		t.Fatal("styles changed table alignment or values")
	}
	for _, char := range plain {
		if char > 127 {
			t.Fatal("ASCII input produced non-ASCII table decorations")
		}
	}
}

func TestCloudStateColorsPreservePlainOutputAtEightyColumns(t *testing.T) {
	data := `[{"name":"ledger","state":"enabled"},{"name":"auth","state":"progressing"},{"name":"payments","state":"degraded"}]`
	styled := render(t, data, presentation.Options{Format: "table", Color: true, Width: 80})
	assertContains(t, styled, "\x1b[32menabled\x1b[0m", "\x1b[33mprogressing\x1b[0m", "\x1b[31mdegraded\x1b[0m")
	plain := render(t, data, presentation.Options{Format: "table", Color: false, Width: 80})
	assertContains(t, plain, "ledger", "auth", "payments", "enabled", "progressing", "degraded")
	if strings.Contains(plain, "\x1b") || ansiPattern.ReplaceAllString(styled, "") != plain {
		t.Fatal("Cloud state colors changed plain text or table alignment")
	}
	assertLineWidths(t, plain, 80)
}

func TestInvalidInputDoesNotWritePartialOutput(t *testing.T) {
	for _, tc := range []struct{ data, format string }{{`{"broken":`, "table"}, {`true false`, "json"}, {`{}`, "yaml"}} {
		var out bytes.Buffer
		if err := presentation.Render(&out, json.RawMessage(tc.data), presentation.Options{Format: tc.format}); err == nil {
			t.Error("accepted invalid JSON or format")
		}
		if out.Len() != 0 {
			t.Fatal("invalid input wrote partial output")
		}
	}
	if err := presentation.Render(nil, json.RawMessage(`{}`), presentation.Options{Format: "json"}); err == nil {
		t.Fatal("accepted nil writer")
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriterErrorsAreReturned(t *testing.T) {
	want := errors.New("output unavailable")
	for _, format := range []string{"json", "table"} {
		if err := presentation.Render(failingWriter{want}, json.RawMessage(`{"ok":true}`), presentation.Options{Format: format}); !errors.Is(err, want) {
			t.Errorf("writer error = %v", err)
		}
	}
}

func TestValidateFormat(t *testing.T) {
	for _, format := range []string{"auto", "json", "table"} {
		if err := presentation.ValidateFormat(format); err != nil {
			t.Fatal(err)
		}
	}
	for _, format := range []string{"", "yaml", "JSON"} {
		if err := presentation.ValidateFormat(format); err == nil {
			t.Errorf("accepted unsupported format %q", format)
		}
	}
}

func TestIndependentRendersCanRunConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if err := presentation.Render(io.Discard, json.RawMessage(`{"data":[{"id":"a"}],"hasMore":true}`), presentation.Options{Format: "table", Color: true}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}
