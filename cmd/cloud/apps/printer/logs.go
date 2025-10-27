package printer

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	"github.com/pterm/pterm"
)

func RenderLogs(out io.Writer, logs []components.Log) error {
	for i, entry := range logs {
		pterm.DefaultSection.Print(fmt.Sprintf("%d - Diagnostic ", i+1))
		data := [][]string{
			{
				"Timestamp",
				entry.Timestamp.String(),
			}, {
				"Severity",
				entry.Diagnostic.Severity,
			}, {
				"Summary",
				entry.Diagnostic.Summary,
			}, {
				"Details",
				entry.Diagnostic.Detail,
			},
		}
		if err := pterm.
			DefaultTable.
			WithData(data).
			WithWriter(out).
			Render(); err != nil {
			return err
		}
	}

	return nil
}
