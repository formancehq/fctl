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

type UpdateQontoConnectorConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type UpdateQontoConnectorConfigController struct {
	PaymentsVersion versions.Version

	store *UpdateQontoConnectorConfigStore

	connectorIDFlag string
}

func (c *UpdateQontoConnectorConfigController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateQontoConnectorConfigStore] = (*UpdateQontoConnectorConfigController)(nil)

func NewUpdateQontoConnectorConfigStore() *UpdateQontoConnectorConfigStore {
	return &UpdateQontoConnectorConfigStore{
		Success: false,
	}
}

func NewUpdateQontoConnectorConfigController() *UpdateQontoConnectorConfigController {
	return &UpdateQontoConnectorConfigController{
		store:           NewUpdateQontoConnectorConfigStore(),
		connectorIDFlag: "connector-id",
	}
}

func newUpdateQontoCommand() *cobra.Command {
	c := NewUpdateQontoConnectorConfigController()
	return fctl.NewCommand(internal.QontoConnector+" <file>|-",
		fctl.WithShortDescription("Update the config of a Qonto connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithController[*UpdateQontoConnectorConfigStore](c),
	)
}

func (c *UpdateQontoConnectorConfigController) GetStore() *UpdateQontoConnectorConfigStore {
	return c.store
}

func (c *UpdateQontoConnectorConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
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

	config := &shared.V3QontoConfig{}
	if err := json.Unmarshal([]byte(script), config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V3.V3UpdateConnectorConfig(cmd.Context(), operations.V3UpdateConnectorConfigRequest{
		V3InstallConnectorRequest: &shared.V3InstallConnectorRequest{
			V3QontoConfig: config,
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

func (c *UpdateQontoConnectorConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' updated!", c.store.ConnectorID)

	return nil
}
