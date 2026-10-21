package presentation

import (
	"fmt"
	"strings"
	"unicode"
)

func escape(text string) string {
	var output strings.Builder
	for _, char := range text {
		var piece string
		switch char {
		case '\n':
			piece = `\n`
		case '\r':
			piece = `\r`
		case '\t':
			piece = `\t`
		default:
			piece = string(char)
			if unicode.IsControl(char) || unicode.Is(unicode.Cf, char) {
				piece = fmt.Sprintf("\\u%04x", char)
			}
		}
		if _, err := output.WriteString(piece); err != nil {
			return ""
		}
	}
	return output.String()
}

// Width calculations operate before adding ANSI styles. Control characters in
// user data have already been escaped, so they cannot introduce terminal codes.
func runeWidth(char rune) int {
	if unicode.Is(unicode.Mn, char) || unicode.Is(unicode.Me, char) {
		return 0
	}
	if char == 0x303f {
		return 1
	}
	for _, interval := range [][2]rune{
		{0x1100, 0x115f}, {0x2329, 0x232a}, {0x2e80, 0xa4cf},
		{0xac00, 0xd7a3}, {0xf900, 0xfaff}, {0xfe10, 0xfe19},
		{0xfe30, 0xfe6f}, {0xff00, 0xff60}, {0xffe0, 0xffe6},
		{0x1f300, 0x1faff}, {0x20000, 0x3fffd},
	} {
		if char >= interval[0] && char <= interval[1] {
			return 2
		}
	}
	return 1
}

func textWidth(text string) int {
	width := 0
	for _, char := range text {
		width += runeWidth(char)
	}
	return width
}

func wrap(text string, width int) []string {
	width = max(width, 1)
	if width == 1 {
		text = narrowText(text)
	}
	var lines []string
	var line strings.Builder
	used := 0
	for _, char := range text {
		size := runeWidth(char)
		if used > 0 && used+size > width {
			lines = append(lines, line.String())
			line.Reset()
			used = 0
		}
		if _, err := line.WriteRune(char); err != nil {
			return nil
		}
		used += size
	}
	return append(lines, line.String())
}

func narrowText(text string) string {
	var output strings.Builder
	for _, char := range text {
		piece := string(char)
		if runeWidth(char) > 1 {
			piece = fmt.Sprintf("\\u%04x", char)
		}
		if _, err := output.WriteString(piece); err != nil {
			return ""
		}
	}
	return output.String()
}

func (r *renderer) table(headers []string, rows [][]string) {
	if r.options.Width < 5*len(headers)+1 {
		r.inlineRows(rows)
		return
	}
	widths := columnWidths(headers, rows, r.options.Width)
	border := tableBorder(widths)
	r.line(border)
	r.tableRow(headers, widths, true)
	r.line(border)
	for _, row := range rows {
		r.tableRow(row, widths, false)
	}
	r.line(border)
}

func (r *renderer) inlineRows(rows [][]string) {
	for _, row := range rows {
		for _, line := range wrap(strings.Join(row, ": "), r.options.Width) {
			r.line(line)
		}
	}
}

func columnWidths(headers []string, rows [][]string, limit int) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = min(limit, max(2, textWidth(header)))
		for _, row := range rows {
			widths[i] = min(limit, max(widths[i], textWidth(row[i])))
		}
	}
	available := limit - (3*len(headers) + 1)
	for totalWidth(widths) > available {
		widest := 0
		for i, width := range widths {
			if width > widths[widest] {
				widest = i
			}
		}
		widths[widest]--
	}
	return widths
}

func totalWidth(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

func tableBorder(widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width+2)
	}
	return "+" + strings.Join(parts, "+") + "+"
}

func (r *renderer) tableRow(row []string, widths []int, header bool) {
	columns := make([][]string, len(widths))
	height := 1
	for i, cell := range row {
		columns[i] = wrap(cell, widths[i])
		height = max(height, len(columns[i]))
	}
	for line := range height {
		cells := make([]string, len(widths))
		for i, width := range widths {
			text := ""
			if line < len(columns[i]) {
				text = columns[i][line]
			}
			padding := strings.Repeat(" ", max(0, width-textWidth(text)))
			if header {
				text = r.style(text, "1;36")
			} else {
				text = r.status(text)
			}
			cells[i] = " " + text + padding + " "
		}
		r.line("|" + strings.Join(cells, "|") + "|")
	}
}
