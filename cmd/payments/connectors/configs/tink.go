package configs

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UpdateTinkConnectorConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type UpdateTinkConnectorConfigController struct {
	store           *UpdateTinkConnectorConfigStore
	connectorIDFlag string
}

var _ fctl.Controller[*UpdateTinkConnectorConfigStore] = (*UpdateTinkConnectorConfigController)(nil)

func NewUpdateTinkConnectorConfigStore() *UpdateTinkConnectorConfigStore {
	return &UpdateTinkConnectorConfigStore{
		Success: false,
	}
}

func NewUpdateTinkConnectorConfigController() *UpdateTinkConnectorConfigController {
	return &UpdateTinkConnectorConfigController{
		store:           NewUpdateTinkConnectorConfigStore(),
		connectorIDFlag: "connector-id",
	}
}

func newUpdateTinkCommand() *cobra.Command {
	c := NewUpdateTinkConnectorConfigController()
	return fctl.NewCommand(internal.TinkConnector+" <file>|-",
		fctl.WithShortDescription("Update the config of a Tink connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithController[*UpdateTinkConnectorConfigStore](c),
	)
}

func (c *UpdateTinkConnectorConfigController) GetStore() *UpdateTinkConnectorConfigStore {
	return c.store
}

func (c *UpdateTinkConnectorConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	config := &shared.V3TinkConfig{}
	if err := json.Unmarshal([]byte(script), config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V3.V3UpdateConnectorConfig(cmd.Context(), operations.V3UpdateConnectorConfigRequest{
		V3InstallConnectorRequest: &shared.V3InstallConnectorRequest{
			V3TinkConfig: config,
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

func (c *UpdateTinkConnectorConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' updated!", c.store.ConnectorID)

	return nil
}
