package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/payments"
)

func DisplayAdyenConfig(cmd *cobra.Command, connectorConfig *payments.ConnectorConfigResponse) error {
	config := connectorConfig.ConnectorConfig.AdyenConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("ApiKey:"), config.APIKey})
	tableData = append(tableData, []string{pterm.LightCyan("HMACKey:"), config.HmacKey})
	tableData = append(tableData, []string{pterm.LightCyan("LiveEndpointPrefix:"), fctl.StringPointerToString(config.LiveEndpointPrefix)})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
