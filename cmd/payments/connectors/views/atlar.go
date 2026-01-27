package views

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

func DisplayAtlarConfig(cmd *cobra.Command, connectorConfig *shared.ConnectorConfigResponse) error {
	config := connectorConfig.Data.AtlarConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("AccessKey:"), config.AccessKey})
	tableData = append(tableData, []string{pterm.LightCyan("Secret:"), config.Secret})
	tableData = append(tableData, []string{pterm.LightCyan("BaseUrl:"), fctl.StringPointerToString(config.BaseURL)})
	tableData = append(tableData, []string{pterm.LightCyan("PageSize:"), func() string {
		if config.PageSize == nil {
			return ""
		}
		return fmt.Sprintf("%d", *config.PageSize)
	}()})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	tableData = append(tableData, []string{pterm.LightCyan("Transfer Initiation Status Polling Period:"), fctl.StringPointerToString(config.TransferInitiationStatusPollingPeriod)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}

func DisplayAtlarConfigV3(cmd *cobra.Command, v3Config *shared.V3GetConnectorConfigResponse) error {
	config := v3Config.Data.V3AtlarConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("AccessKey:"), config.AccessKey})
	tableData = append(tableData, []string{pterm.LightCyan("BaseUrl:"), config.BaseURL})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	tableData = append(tableData, []string{pterm.LightCyan("Secret:"), config.Secret})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
