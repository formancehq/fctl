package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

func DisplayAdyenConfig(cmd *cobra.Command, connectorConfig *shared.ConnectorConfigResponse) error {
	config := connectorConfig.Data.AdyenConfig

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

func DisplayAdyenConfigV3(cmd *cobra.Command, v3Config *shared.V3GetConnectorConfigResponse) error {
	config := v3Config.Data.V3AdyenConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("ApiKey:"), config.APIKey})
	tableData = append(tableData, []string{pterm.LightCyan("CompanyID:"), config.CompanyID})
	tableData = append(tableData, []string{pterm.LightCyan("LiveEndpointPrefix:"), fctl.StringPointerToString(config.LiveEndpointPrefix)})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	tableData = append(tableData, []string{pterm.LightCyan("WebhookPassword:"), fctl.StringPointerToString(config.WebhookPassword)})
	tableData = append(tableData, []string{pterm.LightCyan("WebhookUsername:"), fctl.StringPointerToString(config.WebhookUsername)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
