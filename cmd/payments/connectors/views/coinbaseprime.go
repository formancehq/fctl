package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayCoinbaseprimeConfigV3(cmd *cobra.Command, v3Config *payments.V3GetConnectorConfigResponse) error {
	config := v3Config.V3ConnectorConfig.V3CoinbaseprimeConfig
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
