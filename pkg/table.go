package fctl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/pterm/pterm"
)

// ColumnFit says how a table column gives up width when the table is wider than the
// terminal. Columns are never hidden: identifiers keep FitNone so they stay
// copyable, and the table switches to stacked records when even the minimum
// widths do not fit.
type ColumnFit int

const (
	FitNone ColumnFit = iota
	// FitCut truncates each line with an ellipsis.
	FitCut
	// FitWrap wraps on word boundaries into a multi-line cell.
	FitWrap
)

const ellipsis = "…"

// TableColumn configures one table column. Min defaults to 8 and MaxLines to
// unlimited. Style colours a cell after fitting so widths stay exact.
type TableColumn struct {
	Fit      ColumnFit
	Min      int
	MaxLines int
	Style    func(string) string
}

// TableGroup is a run of table rows under an optional title. All groups of a table
// share the same column widths.
type TableGroup struct {
	Title string
	Rows  [][]string
}

// terminalWidth returns the width available to out, or 0 when out is not the
// interactive standard output. Zero disables fitting so piped output keeps
// every value intact. Tests replace it with SetTerminalWidth.
var terminalWidth = func(out io.Writer) int {
	file, ok := out.(*os.File)
	if !ok || file != os.Stdout {
		return 0
	}
	width, _, err := pterm.GetTerminalSize()
	if err != nil {
		return 0
	}
	return width
}

// SetTerminalWidth forces the width used for every writer and returns a
// function restoring detection. Zero simulates piped output.
func SetTerminalWidth(width int) (restore func()) {
	previous, previousForced := terminalWidth, terminalForced
	terminalWidth = func(io.Writer) int { return width }
	terminalForced = true
	return func() { terminalWidth, terminalForced = previous, previousForced }
}

// terminalForced reports that SetTerminalWidth replaced detection, so help
// also skips its stderr check.
var terminalForced bool

// TerminalWidth returns the width output written to out must fit, or 0 when it
// must not be fitted.
func TerminalWidth(out io.Writer) int {
	return terminalWidth(out)
}

// RenderTable renders a header table fitted to the terminal width.
func RenderTable(out io.Writer, header []string, columns []TableColumn, rows [][]string) error {
	return RenderGroupedTable(out, header, columns, []TableGroup{{Rows: rows}})
}

// RenderGroupedTable renders titled row groups with shared column widths.
func RenderGroupedTable(out io.Writer, header []string, columns []TableColumn, groups []TableGroup) error {
	width := TerminalWidth(out)
	all := make([][]string, 0)
	if header != nil {
		all = append(all, header)
	}
	for _, group := range groups {
		all = append(all, group.Rows...)
	}
	if len(all) == 0 {
		return nil
	}
	widths, fits := fitWidths(all, columns, width)
	if !fits && header != nil {
		return renderStacked(out, header, columns, groups, width)
	}
	if len(groups) == 1 && groups[0].Title == "" {
		data := make(pterm.TableData, 0, len(all))
		for i, row := range all {
			data = append(data, fitRow(row, columns, widths, width, false, header == nil || i > 0))
		}
		if err := renderPTermTable(out, data, header != nil); err != nil {
			return err
		}
		return endBlock(out)
	}
	// One pterm table per group: padding every cell to the shared width keeps
	// the groups aligned even though pterm sizes each table on its own.
	pad := true
	if header != nil {
		cells := fitRow(header, columns, widths, width, pad, false)
		if err := renderPTermTable(out, pterm.TableData{cells}, true); err != nil {
			return err
		}
	}
	for _, group := range groups {
		if group.Title != "" {
			if _, err := fmt.Fprintln(out, pterm.Gray(clip(group.Title, width))); err != nil {
				return err
			}
		}
		if len(group.Rows) == 0 {
			continue
		}
		data := make(pterm.TableData, 0, len(group.Rows))
		for _, row := range group.Rows {
			data = append(data, fitRow(row, columns, widths, width, pad, true))
		}
		if err := renderPTermTable(out, data, false); err != nil {
			return err
		}
	}
	return endBlock(out)
}

// RenderKeyValues renders label/value rows, wrapping values to the terminal.
func RenderKeyValues(out io.Writer, rows [][]string) error {
	columns := []TableColumn{{Style: lightCyan}, {Fit: FitWrap, Min: 16, MaxLines: 6}}
	width := TerminalWidth(out)
	widths, fits := fitWidths(rows, columns, width)
	if !fits {
		return renderStackedKeyValues(out, rows, columns[1], width)
	}
	data := make(pterm.TableData, 0, len(rows))
	for _, row := range rows {
		data = append(data, fitRow(row, columns, widths, width, false, true))
	}
	if err := renderPTermTable(out, data, false); err != nil {
		return err
	}
	return endBlock(out)
}

// renderStackedKeyValues prints each label on its own line with the value
// wrapped and indented below it, for terminals too narrow for two columns.
func renderStackedKeyValues(out io.Writer, rows [][]string, value TableColumn, width int) error {
	valueWidth := max(1, width-2)
	var builder strings.Builder
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		builder.WriteString(lightCyan(row[0]) + "\n")
		if len(row) < 2 {
			continue
		}
		for _, line := range clampLines(Wrap(row[1], valueWidth), value.MaxLines, valueWidth) {
			builder.WriteString("  " + line + "\n")
		}
	}
	builder.WriteString("\n")
	_, err := io.WriteString(out, builder.String())
	return err
}

// RenderText writes text wrapped to the terminal width.
func RenderText(out io.Writer, text string) error {
	lines := []string{text}
	if width := TerminalWidth(out); width > 0 {
		lines = Wrap(text, width)
	}
	_, err := fmt.Fprintln(out, strings.Join(lines, "\n"))
	return err
}

// RenderBullet writes "- text" with wrapped lines indented under the text.
func RenderBullet(out io.Writer, text string) error {
	lines := []string{text}
	if width := TerminalWidth(out); width > 2 {
		lines = Wrap(text, width-2)
	}
	_, err := fmt.Fprintln(out, "- "+strings.Join(lines, "\n  "))
	return err
}

// RenderNextPage prints the cursor for the next page on its own line. The
// token is never truncated because it must be copied into --cursor.
func RenderNextPage(out io.Writer, cursor Cursor) error {
	if !cursor.HasMore || cursor.Next == nil || *cursor.Next == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "\n%s--cursor %s\n", pterm.Gray("Next page: "), *cursor.Next)
	return err
}

// Truncate shortens s to width display cells, ending with an ellipsis. Wide
// runes such as CJK count as two cells.
func Truncate(s string, width int) string {
	if width <= 0 || textWidth(s) <= width {
		return s
	}
	// Cutting through an escape sequence would corrupt the terminal, so
	// coloured input loses its colours when it has to be shortened.
	return runewidth.Truncate(pterm.RemoveColorFromString(s), width, ellipsis)
}

// Wrap splits s into lines of at most width cells on word boundaries,
// hard-breaking words longer than a line.
func Wrap(s string, width int) []string {
	if width <= 0 {
		return strings.Split(s, "\n")
	}
	if textWidth(s) != runewidth.StringWidth(s) {
		s = pterm.RemoveColorFromString(s) // see Truncate
	}
	lines := make([]string, 0)
	for _, paragraph := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			for textWidth(word) > width {
				if line != "" {
					lines = append(lines, line)
					line = ""
				}
				head, rest := splitAtWidth(word, width)
				if rest == "" {
					// A single rune wider than the line: keep it as the word.
					word = head
					break
				}
				lines = append(lines, head)
				word = rest
			}
			switch {
			case word == "":
			case line == "":
				line = word
			case textWidth(line)+1+textWidth(word) <= width:
				line += " " + word
			default:
				lines = append(lines, line)
				line = word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// textWidth is the number of terminal cells s occupies once colours are
// removed, the same measure pterm uses to align columns.
func textWidth(s string) int {
	return runewidth.StringWidth(pterm.RemoveColorFromString(s))
}

// splitAtWidth cuts s after at most width cells, always keeping at least one
// rune in the head so a rune wider than width still makes progress.
func splitAtWidth(s string, width int) (string, string) {
	cells := 0
	for index, r := range s {
		next := runewidth.RuneWidth(r)
		if cells+next > width {
			if index == 0 {
				size := len(string(r))
				return s[:size], s[size:]
			}
			return s[:index], s[index:]
		}
		cells += next
	}
	return s, ""
}

func clip(s string, width int) string {
	if width <= 0 {
		return s
	}
	return Truncate(s, width)
}

func tableColumn(columns []TableColumn, index int) TableColumn {
	if index < len(columns) {
		return columns[index]
	}
	return TableColumn{}
}

func minimumWidth(c TableColumn) int {
	if c.Min > 0 {
		return c.Min
	}
	return 8
}

// fitWidths shrinks the widest flexible column one cell at a time until the
// table fits. It reports false when the minimum widths still overflow.
func fitWidths(rows [][]string, columns []TableColumn, width int) ([]int, bool) {
	count := 0
	for _, row := range rows {
		count = max(count, len(row))
	}
	widths := make([]int, count)
	for _, row := range rows {
		for i, cell := range row {
			for _, line := range strings.Split(cell, "\n") {
				widths[i] = max(widths[i], textWidth(line))
			}
		}
	}
	if width <= 0 {
		return widths, true
	}
	total := 3 * (count - 1)
	for _, w := range widths {
		total += w
	}
	for total > width {
		best := -1
		for i := range widths {
			c := tableColumn(columns, i)
			if c.Fit != FitNone && widths[i] > minimumWidth(c) && (best < 0 || widths[i] > widths[best]) {
				best = i
			}
		}
		if best < 0 {
			return widths, false
		}
		widths[best]--
		total--
	}
	return widths, true
}

func fitCell(cell string, c TableColumn, width int) []string {
	lines := strings.Split(cell, "\n")
	switch {
	case width <= 0 || c.Fit == FitNone:
	case c.Fit == FitCut:
		for i, line := range lines {
			lines[i] = Truncate(line, width)
		}
	case c.Fit == FitWrap:
		lines = clampLines(Wrap(cell, width), c.MaxLines, width)
	}
	return lines
}

func clampLines(lines []string, maxLines, width int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	lines = lines[:maxLines]
	last := lines[maxLines-1]
	if textWidth(last)+2 <= width {
		lines[maxLines-1] = last + " " + ellipsis
	} else {
		lines[maxLines-1] = Truncate(last+ellipsis+ellipsis, width)
	}
	return lines
}

func fitRow(row []string, columns []TableColumn, widths []int, terminal int, pad, styled bool) []string {
	cells := make([]string, len(row))
	for i, cell := range row {
		c := tableColumn(columns, i)
		lineWidth := 0
		if terminal > 0 {
			lineWidth = widths[i]
		}
		lines := fitCell(cell, c, lineWidth)
		for j, line := range lines {
			if pad && i < len(row)-1 {
				line += strings.Repeat(" ", max(0, widths[i]-textWidth(line)))
			}
			if styled && c.Style != nil && line != "" {
				line = c.Style(line)
			}
			lines[j] = line
		}
		cells[i] = strings.Join(lines, "\n")
	}
	return cells
}

// renderPTermTable writes the table without the blank line pterm's Render
// appends, so grouped tables stay contiguous; endBlock restores that blank line
// once the whole block is written, keeping today's spacing between sections.
func renderPTermTable(out io.Writer, data pterm.TableData, header bool) error {
	rendered, err := pterm.DefaultTable.WithHasHeader(header).WithData(data).Srender()
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, strings.TrimSuffix(rendered, "\n")+"\n")
	return err
}

func endBlock(out io.Writer) error {
	_, err := io.WriteString(out, "\n")
	return err
}

// renderStacked prints one block per row when the table cannot fit: the first
// cell as a title, then the remaining non-empty cells as labelled lines.
func renderStacked(out io.Writer, header []string, columns []TableColumn, groups []TableGroup, width int) error {
	labelWidth := 0
	for _, label := range header[1:] {
		labelWidth = max(labelWidth, textWidth(label))
	}
	// Labels sit beside their value when that leaves a readable value column;
	// otherwise each label gets its own line and the value wraps below it.
	const minimumStackedValue = 10
	inline := width-4-labelWidth >= minimumStackedValue
	valueWidth, valueIndent := width-4-labelWidth, "  "+strings.Repeat(" ", labelWidth)+"  "
	if !inline {
		valueWidth, valueIndent = max(1, width-4), "    "
	}
	var builder strings.Builder
	first := true
	for _, group := range groups {
		if group.Title != "" {
			if !first {
				builder.WriteString("\n")
			}
			builder.WriteString(pterm.Gray(clip(group.Title, width)) + "\n")
			first = true
		}
		for _, row := range group.Rows {
			if len(row) == 0 {
				continue
			}
			if !first {
				builder.WriteString("\n")
			}
			first = false
			builder.WriteString(pterm.Bold.Sprint(row[0]) + "\n")
			for i := 1; i < len(row) && i < len(header); i++ {
				if row[i] == "" {
					continue
				}
				c := tableColumn(columns, i)
				// Identifiers stay whole so they can be copied, even if the
				// terminal soft-wraps them.
				lines := strings.Split(row[i], "\n")
				if c.Fit != FitNone {
					lines = clampLines(Wrap(row[i], valueWidth), c.MaxLines, valueWidth)
				}
				if !inline {
					builder.WriteString("  " + pterm.LightCyan(header[i]) + "\n")
				}
				for j, line := range lines {
					if c.Style != nil {
						line = c.Style(line)
					}
					prefix := valueIndent
					if inline && j == 0 {
						prefix = "  " + pterm.LightCyan(header[i]+strings.Repeat(" ", labelWidth-textWidth(header[i]))) + "  "
					}
					builder.WriteString(prefix + line + "\n")
				}
			}
		}
	}
	builder.WriteString("\n")
	_, err := io.WriteString(out, builder.String())
	return err
}

func lightCyan(s string) string { return pterm.LightCyan(s) }
