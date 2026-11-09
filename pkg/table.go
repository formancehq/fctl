package fctl

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

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
	previous := terminalWidth
	terminalWidth = func(io.Writer) int { return width }
	return func() { terminalWidth = previous }
}

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
	widths, _ := fitWidths(rows, columns, width)
	data := make(pterm.TableData, 0, len(rows))
	for _, row := range rows {
		data = append(data, fitRow(row, columns, widths, width, false, true))
	}
	if err := renderPTermTable(out, data, false); err != nil {
		return err
	}
	return endBlock(out)
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

// Truncate shortens s to width display cells, ending with an ellipsis.
func Truncate(s string, width int) string {
	if width <= 0 || textWidth(s) <= width {
		return s
	}
	if width == 1 {
		return ellipsis
	}
	return string([]rune(s)[:width-1]) + ellipsis
}

// Wrap splits s into lines of at most width cells on word boundaries,
// hard-breaking words longer than a line.
func Wrap(s string, width int) []string {
	if width <= 0 {
		return strings.Split(s, "\n")
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
				runes := []rune(word)
				lines = append(lines, string(runes[:width]))
				word = string(runes[width:])
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

func textWidth(s string) int {
	return utf8.RuneCountInString(pterm.RemoveColorFromString(s))
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
	valueWidth := max(10, width-4-labelWidth)
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
				for j, line := range clampLines(Wrap(row[i], valueWidth), c.MaxLines, valueWidth) {
					label := strings.Repeat(" ", labelWidth)
					if j == 0 {
						label = pterm.LightCyan(header[i] + strings.Repeat(" ", labelWidth-textWidth(header[i])))
					}
					if c.Style != nil {
						line = c.Style(line)
					}
					builder.WriteString("  " + label + "  " + line + "\n")
				}
			}
		}
	}
	builder.WriteString("\n")
	_, err := io.WriteString(out, builder.String())
	return err
}

func lightCyan(s string) string { return pterm.LightCyan(s) }
