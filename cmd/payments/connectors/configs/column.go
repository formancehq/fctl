package configs

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type UpdateColumnConnectorConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type UpdateColumnConnectorConfigController struct {
	PaymentsVersion versions.Version

	store *UpdateColumnConnectorConfigStore

	connectorIDFlag string
}

func (c *UpdateColumnConnectorConfigController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateColumnConnectorConfigStore] = (*UpdateColumnConnectorConfigController)(nil)

func NewUpdateColumnConnectorConfigStore() *UpdateColumnConnectorConfigStore {
	return &UpdateColumnConnectorConfigStore{
		Success: false,
	}
}

func NewUpdateColumnConnectorConfigController() *UpdateColumnConnectorConfigController {
	return &UpdateColumnConnectorConfigController{
		store:           NewUpdateColumnConnectorConfigStore(),
		connectorIDFlag: "connector-id",
	}
}

func newUpdateColumnCommand() *cobra.Command {
	c := NewUpdateColumnConnectorConfigController()
	return fctl.NewCommand(internal.ColumnConnector+" <file>|-",
		fctl.WithShortDescription("Update the config of a Column connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithController[*UpdateColumnConnectorConfigStore](c),
	)
}

func (c *UpdateColumnConnectorConfigController) GetStore() *UpdateColumnConnectorConfigStore {
	return c.store
}

func (c *UpdateColumnConnectorConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	connectorID := fctl.GetString(cmd, c.connectorIDFlag)
	if connectorID == "" {
		return nil, fmt.Errorf("missing connector ID")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to update the config of connector '%s'", connectorID) {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	config := &shared.V3ColumnConfig{}
	if err := json.Unmarshal([]byte(script), config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V3.V3UpdateConnectorConfig(cmd.Context(), operations.V3UpdateConnectorConfigRequest{
		V3InstallConnectorRequest: &shared.V3InstallConnectorRequest{
			V3ColumnConfig: config,
		},
		ConnectorID: connectorID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating config of connector: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = true
	c.store.ConnectorID = connectorID

	return c, nil
}

func (c *UpdateColumnConnectorConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' updated!", c.store.ConnectorID)

	return nil
}
