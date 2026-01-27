package views

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

func DisplayBankingCircleConfig(cmd *cobra.Command, connectorConfig *shared.ConnectorConfigResponse) error {
	config := connectorConfig.Data.BankingCircleConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("Username:"), config.Username})
	tableData = append(tableData, []string{pterm.LightCyan("Password:"), config.Password})
	tableData = append(tableData, []string{pterm.LightCyan("Endpoint:"), config.Endpoint})
	tableData = append(tableData, []string{pterm.LightCyan("Authorization endpoint:"), config.AuthorizationEndpoint})
	tableData = append(tableData, []string{pterm.LightCyan("UserCertificate:"), config.UserCertificate})
	tableData = append(tableData, []string{pterm.LightCyan("UserCertificateKey:"), config.UserCertificateKey})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}

func DisplayBankingCircleConfigV3(cmd *cobra.Command, v3Config *shared.V3GetConnectorConfigResponse) error {
	config := v3Config.Data.V3BankingcircleConfig

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Name:"), config.Name})
	tableData = append(tableData, []string{pterm.LightCyan("Username:"), config.Username})
	tableData = append(tableData, []string{pterm.LightCyan("Password:"), config.Password})
	tableData = append(tableData, []string{pterm.LightCyan("Endpoint:"), config.Endpoint})
	tableData = append(tableData, []string{pterm.LightCyan("Authorization endpoint:"), config.AuthorizationEndpoint})
	tableData = append(tableData, []string{pterm.LightCyan("Polling Period:"), fctl.StringPointerToString(config.PollingPeriod)})
	tableData = append(tableData, []string{pterm.LightCyan("UserCertificate:"), config.UserCertificate})
	tableData = append(tableData, []string{pterm.LightCyan("UserCertificateKey:"), config.UserCertificateKey})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return nil
}
