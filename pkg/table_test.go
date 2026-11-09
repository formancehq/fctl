package fctl

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

func plainLines(t *testing.T, output string) []string {
	t.Helper()
	return strings.Split(strings.TrimRight(pterm.RemoveColorFromString(output), "\n"), "\n")
}

func requireFits(t *testing.T, output string, width int) {
	t.Helper()
	for _, line := range plainLines(t, output) {
		require.LessOrEqual(t, utf8.RuneCountInString(strings.TrimRight(line, " ")), width, "line %q exceeds %d columns", line, width)
	}
}

func withWidth(t *testing.T, width int) {
	t.Helper()
	t.Cleanup(SetTerminalWidth(width))
}

var listHeader = []string{"Name", "Connector", "Ledger", "Target", "Applied", "Status"}
var listColumns = []TableColumn{{}, {Fit: FitCut, Min: 10}, {Fit: FitCut, Min: 10}, {}, {}, {}}
var listRows = [][]string{{"stripeconnect-fpss", "ee-stripeconnect", "stripeconnect-fpss", "beta", "v0.5.0-beta.14", "active"}}

func TestTruncateKeepsShortValuesAndEndsLongOnesWithEllipsis(t *testing.T) {
	require.Equal(t, "stripe", Truncate("stripe", 10))
	require.Equal(t, "ee-stripe…", Truncate("ee-stripeconnect", 10))
	require.Equal(t, "…", Truncate("abc", 1))
	require.Equal(t, "abc", Truncate("abc", 0))
}

func TestWrapBreaksOnWordsAndHardBreaksLongWords(t *testing.T) {
	require.Equal(t, []string{"Restricted API", "key."}, Wrap("Restricted API key.", 14))
	require.Equal(t, []string{"abcd", "efgh", "ij"}, Wrap("abcdefghij", 4))
	require.Equal(t, []string{"one", "", "two"}, Wrap("one\n\ntwo", 10))
}

func TestRenderTableGivenWideTerminalWhenRenderedThenNothingIsCut(t *testing.T) {
	withWidth(t, 120)
	var out bytes.Buffer

	require.NoError(t, RenderTable(&out, listHeader, listColumns, listRows))

	require.Contains(t, out.String(), "ee-stripeconnect")
	require.NotContains(t, out.String(), "…")
	requireFits(t, out.String(), 120)
}

func TestRenderTableGivenEightyColumnsWhenTooWideThenFlexibleColumnsShrinkAndIdentifiersStay(t *testing.T) {
	withWidth(t, 80)
	var out bytes.Buffer

	require.NoError(t, RenderTable(&out, listHeader, listColumns, listRows))

	output := pterm.RemoveColorFromString(out.String())
	require.Contains(t, output, "stripeconnect-fpss |", "the name column must stay intact")
	require.Contains(t, output, "v0.5.0-beta.14")
	require.Contains(t, output, "…")
	requireFits(t, out.String(), 80)
}

func TestRenderTableGivenNarrowTerminalWhenMinimumsOverflowThenRowsAreStacked(t *testing.T) {
	withWidth(t, 40)
	var out bytes.Buffer

	require.NoError(t, RenderTable(&out, listHeader, listColumns, listRows))

	lines := plainLines(t, out.String())
	require.Equal(t, "stripeconnect-fpss", lines[0])
	require.Contains(t, lines, "  Connector  ee-stripeconnect")
	require.Contains(t, lines, "  Applied    v0.5.0-beta.14")
	requireFits(t, out.String(), 40)
}

func TestRenderTableGivenPipedOutputWhenRenderedThenValuesAreUntouched(t *testing.T) {
	withWidth(t, 0)
	long := strings.Repeat("x", 200)
	var out bytes.Buffer

	require.NoError(t, RenderTable(&out, []string{"Name", "Value"}, []TableColumn{{}, {Fit: FitWrap}}, [][]string{{"a", long}}))

	require.Contains(t, out.String(), long)
}

func TestRenderGroupedTableAlignsGroupsAndPrintsTitles(t *testing.T) {
	withWidth(t, 0)
	var out bytes.Buffer
	groups := []TableGroup{
		{Title: "— Authentication", Rows: [][]string{{"SECRET_KEY", "yes"}}},
		{Title: "— Limits (advanced)", Rows: [][]string{{"MAX_RETRIES", ""}}},
	}

	require.NoError(t, RenderGroupedTable(&out, []string{"Key", "Req"}, nil, groups))

	lines := plainLines(t, out.String())
	require.Equal(t, []string{"Key         | Req", "— Authentication", "SECRET_KEY  | yes", "— Limits (advanced)", "MAX_RETRIES |"}, trimRight(lines))
}

func TestRenderTableWrapsAndClampsDescriptions(t *testing.T) {
	withWidth(t, 40)
	var out bytes.Buffer
	description := strings.Repeat("word ", 40)

	require.NoError(t, RenderTable(&out, []string{"Key", "Description"}, []TableColumn{{}, {Fit: FitWrap, Min: 20, MaxLines: 3}}, [][]string{{"KEY", description}}))

	lines := plainLines(t, out.String())
	require.Len(t, lines, 4, "header plus three description lines")
	require.True(t, strings.HasSuffix(strings.TrimRight(lines[3], " "), "…"))
	requireFits(t, out.String(), 40)
}

func TestRenderKeyValuesWrapsValuesUnderTheValueColumn(t *testing.T) {
	withWidth(t, 40)
	var out bytes.Buffer

	require.NoError(t, RenderKeyValues(&out, [][]string{{"Description", strings.Repeat("long text ", 10)}, {"Name", "ee-base"}}))

	lines := trimRight(plainLines(t, out.String()))
	require.True(t, strings.HasPrefix(lines[0], "Description | long text"))
	require.True(t, strings.HasPrefix(lines[1], "            | text long"))
	require.Equal(t, "Name        | ee-base", lines[4])
	requireFits(t, out.String(), 40)
}

func TestRenderTextWrapsOnlyInATerminal(t *testing.T) {
	message := `Connection "stripeconnect-fpss" installed with connector "ee-stripeconnect" for ledger "stripeconnect-fpss".`
	withWidth(t, 50)
	var out bytes.Buffer
	require.NoError(t, RenderText(&out, message))
	require.Greater(t, len(plainLines(t, out.String())), 1)
	requireFits(t, out.String(), 50)

	restore := SetTerminalWidth(0)
	defer restore()
	out.Reset()
	require.NoError(t, RenderText(&out, message))
	require.Equal(t, message+"\n", out.String())
}

func TestRenderNextPagePrintsTheFullCursorOnlyWhenMoreResultsExist(t *testing.T) {
	withWidth(t, 20)
	token := strings.Repeat("Q", 60)
	var out bytes.Buffer

	require.NoError(t, RenderNextPage(&out, Cursor{HasMore: true, Next: &token}))
	require.Contains(t, pterm.RemoveColorFromString(out.String()), "Next page: --cursor "+token)

	out.Reset()
	require.NoError(t, RenderNextPage(&out, Cursor{HasMore: false, Next: &token}))
	require.Empty(t, out.String())
}

func TestRenderBulletIndentsWrappedLines(t *testing.T) {
	withWidth(t, 24)
	var out bytes.Buffer

	require.NoError(t, RenderBullet(&out, "Choose exactly one authentication method."))

	require.Equal(t, []string{"- Choose exactly one", "  authentication method."}, plainLines(t, out.String()))
}

func TestTerminalWidthIsZeroForWritersOtherThanStdout(t *testing.T) {
	require.Zero(t, TerminalWidth(&bytes.Buffer{}))
}

func trimRight(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimRight(line, " ")
	}
	return out
}

func TestTruncateAndWrapCountWideRunesAsTwoCells(t *testing.T) {
	require.Equal(t, "日本…", Truncate("日本語の説明", 5))
	require.Equal(t, []string{"日本", "語"}, Wrap("日本語", 4))
	require.Equal(t, []string{"日", "本"}, Wrap("日本", 1), "a rune wider than the line still makes progress")
}

func TestRenderTableWithWideRunesFitsTheTerminal(t *testing.T) {
	withWidth(t, 30)
	var out bytes.Buffer

	require.NoError(t, RenderTable(&out, []string{"Name", "Tagline"}, []TableColumn{{}, {Fit: FitCut, Min: 6}}, [][]string{{"ee-jp", "日本の銀行口座の入出金を取り込みます"}}))

	for _, line := range plainLines(t, out.String()) {
		require.LessOrEqual(t, textWidth(strings.TrimRight(line, " ")), 30, "line %q", line)
	}
}

func TestRenderKeyValuesGivenTooNarrowTerminalThenStacksLabelsAndValues(t *testing.T) {
	withWidth(t, 24)
	var out bytes.Buffer

	require.NoError(t, RenderKeyValues(&out, [][]string{{"Available Versions", "37"}, {"Description", "Ingest connected-account topology."}}))

	lines := plainLines(t, out.String())
	require.Equal(t, []string{"Available Versions", "  37", "Description", "  Ingest", "  connected-account", "  topology."}, lines)
}

func TestRenderTableStackedKeepsIdentifierColumnsWhole(t *testing.T) {
	withWidth(t, 24)
	var out bytes.Buffer
	version := "v0.5.0-beta.14+build.20261109"

	require.NoError(t, RenderTable(&out, []string{"Name", "Applied", "Tagline"}, []TableColumn{{}, {}, {Fit: FitWrap, Min: 30}}, [][]string{{"stripeconnect-fpss", version, "Stripe marketplace payments"}}))

	require.Contains(t, plainLines(t, out.String()), "  Applied  "+version)
}

func TestTruncateAndWrapNeverCutThroughColourCodes(t *testing.T) {
	coloured := "\x1b[31mabcdef\x1b[0m"

	require.Equal(t, "abc…", Truncate(coloured, 4))
	require.Equal(t, []string{"abc", "def"}, Wrap(coloured, 3))
}

func TestRenderTableStackedSkipsEmptyRows(t *testing.T) {
	withWidth(t, 10)
	var out bytes.Buffer

	require.NotPanics(t, func() {
		require.NoError(t, RenderTable(&out, []string{"Name", "Value"}, nil, [][]string{{}, {"a-very-long-name", "value"}}))
	})
	require.Contains(t, out.String(), "a-very-long-name")
}

func TestHelpWidthIsZeroWhenStderrIsNotATerminal(t *testing.T) {
	require.Zero(t, helpWidth(), "go test does not attach stderr to a terminal")
}
