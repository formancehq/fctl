package views

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/wallets"
)

func PrintHold(out io.Writer, hold wallets.ExpandedDebitHold) error {
	fctl.Section.Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), fmt.Sprint(hold.ID)})
	tableData = append(tableData, []string{pterm.LightCyan("Wallet ID"), hold.WalletID})
	tableData = append(tableData, []string{pterm.LightCyan("Original amount"), fmt.Sprint(hold.OriginalAmount)})
	tableData = append(tableData, []string{pterm.LightCyan("Remaining"), fmt.Sprint(hold.Remaining)})

	if err := pterm.DefaultTable.
		WithWriter(out).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return fctl.PrintMetadata(out, hold.Metadata)
}
