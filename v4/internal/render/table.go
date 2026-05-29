package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func Table(writer io.Writer, headers []string, rows [][]string) error {
	rendered := table.New().
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderHeader(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return Styles.TableHeader
			}
			return Styles.TableCell
		}).
		String()
	_, err := fmt.Fprintln(writer, rendered)
	return err
}

func KeyValues(writer io.Writer, rows [][]string) error {
	width := 0
	for _, row := range rows {
		if len(row) > 0 && len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		key := row[0] + strings.Repeat(" ", width-len(row[0]))
		if _, err := fmt.Fprintf(writer, "%s │ %s\n", Styles.TableHeader.Render(key), row[1]); err != nil {
			return err
		}
	}
	return nil
}
