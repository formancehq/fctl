package views

import (
	"io"

	"github.com/pterm/pterm"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayAccounts(out io.Writer, accounts []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, account := range accounts {
		tableData = append(tableData, []string{
			// TODO: Missing property 'ledger' on api response
			//account["ledger"].(string),
			account["address"].(string),
		})
	}
	tableData = fctl.Prepend(tableData, []string{ /*"Ledger",*/ "Address"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}
