package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayPlaidConfigV3(cmd *cobra.Command, v3Config *payments.V3GetConnectorConfigResponse) error {
	config := v3Config.V3ConnectorConfig.V3PlaidConfig
	if config == nil {
		return nil
	}

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("ClientID:"), config.ClientID})
	tableData = append(tableData, []string{pterm.LightCyan("ClientSecret:"), config.ClientSecret})
	tableData = append(tableData, []string{pterm.LightCyan("IsSandbox:"), fctl.BoolPointerToString(config.IsSandbox)})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
