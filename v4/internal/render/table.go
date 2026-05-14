package render

import (
	"io"

	"github.com/pterm/pterm"
)

func Table(writer io.Writer, headers []string, rows [][]string) error {
	data := make(pterm.TableData, 0, len(rows)+1)
	header := make([]string, 0, len(headers))
	for _, value := range headers {
		header = append(header, pterm.LightCyan(value))
	}
	data = append(data, header)
	data = append(data, rows...)
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(writer).
		WithData(data).
		Render()
}
