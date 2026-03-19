package views

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayAssets(out io.Writer, assets []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, asset := range assets {
		tableData = append(tableData, []string{
			asset["ledger"].(string),
			asset["name"].(string),
			asset["account"].(string),
			fmt.Sprint(asset["input"].(float64)),
			fmt.Sprint(asset["output"].(float64)),
		})
	}
	tableData = fctl.Prepend(tableData, []string{"Ledger", "Asset", "Account", "Input", "Output"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}
