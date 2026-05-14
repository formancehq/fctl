package render

import (
	"fmt"
	"io"

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
