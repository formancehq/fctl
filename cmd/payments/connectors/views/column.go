package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

// Column is a connector implemented in v3, so not compatible with the v1 call.

func DisplayColumnConfigV3(cmd *cobra.Command, v3Config *payments.V3GetConnectorConfigResponse) error {
	config := v3Config.V3ConnectorConfig.V3ColumnConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("APIKey:"), config.APIKey})
	tableData = append(tableData, []string{pterm.LightCyan("Endpoint:"), config.Endpoint})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
