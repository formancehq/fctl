package views

import (
	"encoding/json"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func DisplayQontoConfig(cmd *cobra.Command, connectorConfig *shared.ConnectorConfigResponse) error {
	// Display raw JSON since SDK might not have QontoConfig type yet
	jsonData, err := json.MarshalIndent(connectorConfig.Data, "", "  ")
	if err != nil {
		return err
	}
	pterm.DefaultBox.WithWriter(cmd.OutOrStdout()).WithTitle("Qonto Configuration").Println(string(jsonData))
	return nil
}
