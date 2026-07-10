package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
)

func DisplayMangopayConfig(cmd *cobra.Command, connectorConfig *payments.ConnectorConfigResponse) error {
	config := connectorConfig.ConnectorConfig.MangoPayConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("ClientID:"), config.ClientID})
	tableData = append(tableData, []string{pterm.LightCyan("API key:"), config.APIKey})
	tableData = append(tableData, []string{pterm.LightCyan("Endpoint:"), config.Endpoint})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), func() string {
		if config.PollingPeriod == nil {
			return ""
		}
		return *config.PollingPeriod
	}()})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
