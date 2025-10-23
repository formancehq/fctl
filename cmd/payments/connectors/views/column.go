package views

import (
	"encoding/json"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func DisplayColumnConfig(cmd *cobra.Command, connectorConfig *shared.ConnectorConfigResponse) error {
	// Display raw JSON since SDK might not have ColumnConfig type yet
	jsonData, err := json.MarshalIndent(connectorConfig.Data, "", "  ")
	if err != nil {
		return err
	}
	pterm.DefaultBox.WithWriter(cmd.OutOrStdout()).WithTitle("Column Configuration").Println(string(jsonData))
	return nil
}
