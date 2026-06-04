package views

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/wallets"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func PrintHold(out io.Writer, hold wallets.Hold1) error {
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
