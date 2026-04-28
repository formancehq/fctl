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

type UpdateCoinbaseprimeConnectorConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type UpdateCoinbaseprimeConnectorConfigController struct {
	store           *UpdateCoinbaseprimeConnectorConfigStore
	connectorIDFlag string
}

var _ fctl.Controller[*UpdateCoinbaseprimeConnectorConfigStore] = (*UpdateCoinbaseprimeConnectorConfigController)(nil)

func NewUpdateCoinbaseprimeConnectorConfigStore() *UpdateCoinbaseprimeConnectorConfigStore {
	return &UpdateCoinbaseprimeConnectorConfigStore{
		Success: false,
	}
}

func NewUpdateCoinbaseprimeConnectorConfigController() *UpdateCoinbaseprimeConnectorConfigController {
	return &UpdateCoinbaseprimeConnectorConfigController{
		store:           NewUpdateCoinbaseprimeConnectorConfigStore(),
		connectorIDFlag: "connector-id",
	}
}

func newUpdateCoinbaseprimeCommand() *cobra.Command {
	c := NewUpdateCoinbaseprimeConnectorConfigController()
	return fctl.NewCommand(internal.CoinbaseprimeConnector+" <file>|-",
		fctl.WithShortDescription("Update the config of a Coinbase Prime connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithController[*UpdateCoinbaseprimeConnectorConfigStore](c),
	)
}

func (c *UpdateCoinbaseprimeConnectorConfigController) GetStore() *UpdateCoinbaseprimeConnectorConfigStore {
	return c.store
}

func (c *UpdateCoinbaseprimeConnectorConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	config := &shared.V3CoinbaseprimeConfig{}
	if err := json.Unmarshal([]byte(script), config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V3.V3UpdateConnectorConfig(cmd.Context(), operations.V3UpdateConnectorConfigRequest{
		V3InstallConnectorRequest: &shared.V3InstallConnectorRequest{
			V3CoinbaseprimeConfig: config,
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

func (c *UpdateCoinbaseprimeConnectorConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' updated!", c.store.ConnectorID)

	return nil
}
