package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayCoinbaseprimeConfigV3(cmd *cobra.Command, v3Config *shared.V3GetConnectorConfigResponse) error {
	config := v3Config.Data.V3CoinbaseprimeConfig
	if config == nil {
		return nil
	}

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("APIKey:"), config.APIKey})
	tableData = append(tableData, []string{pterm.LightCyan("APISecret:"), config.APISecret})
	tableData = append(tableData, []string{pterm.LightCyan("Passphrase:"), config.Passphrase})
	tableData = append(tableData, []string{pterm.LightCyan("PortfolioID:"), config.PortfolioID})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
