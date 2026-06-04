package views

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func DisplayPowensConfigV3(cmd *cobra.Command, v3Config *payments.V3GetConnectorConfigResponse) error {
	config := v3Config.V3ConnectorConfig.V3PowensConfig
	if config == nil {
		return nil
	}

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("ClientID:"), config.ClientID})
	tableData = append(tableData, []string{pterm.LightCyan("ClientSecret:"), config.ClientSecret})
	tableData = append(tableData, []string{pterm.LightCyan("ConfigurationToken:"), config.ConfigurationToken})
	tableData = append(tableData, []string{pterm.LightCyan("Domain:"), config.Domain})
	tableData = append(tableData, []string{pterm.LightCyan("Endpoint:"), config.Endpoint})
	tableData = append(tableData, []string{pterm.LightCyan("MaxConnectionsPerLink:"), fmt.Sprintf("%d", config.MaxConnectionsPerLink)})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
