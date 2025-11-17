package views

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	fctl "github.com/formancehq/fctl/pkg"
)

func DisplayPayments(out io.Writer, payments []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, payment := range payments {
		tableData = append(tableData, []string{
			payment["provider"].(string),
			payment["reference"].(string),
			fmt.Sprint(payment["amount"].(float64)),
			payment["asset"].(string),
			payment["createdAt"].(string),
			payment["scheme"].(string),
			payment["status"].(string),
			payment["type"].(string),
		})
	}
	tableData = fctl.Prepend(tableData, []string{"Provider", "Reference", "Account",
		"Asset", "Created at", "Scheme", "Status", "Type"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}
