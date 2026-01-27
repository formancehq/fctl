package configs

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type UpdateMoneycorpConnectorConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type UpdateMoneycorpConnectorConfigController struct {
	PaymentsVersion versions.Version

	store *UpdateMoneycorpConnectorConfigStore

	connectorIDFlag string
}

func (c *UpdateMoneycorpConnectorConfigController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateMoneycorpConnectorConfigStore] = (*UpdateMoneycorpConnectorConfigController)(nil)

func NewUpdateMoneycorpConnectorConfigStore() *UpdateMoneycorpConnectorConfigStore {
	return &UpdateMoneycorpConnectorConfigStore{
		Success: false,
	}
}

func NewUpdateMoneycorpConnectorConfigController() *UpdateMoneycorpConnectorConfigController {
	return &UpdateMoneycorpConnectorConfigController{
		store:           NewUpdateMoneycorpConnectorConfigStore(),
		connectorIDFlag: "connector-id",
	}
}

func newUpdateMoneycorpCommand() *cobra.Command {
	c := NewUpdateMoneycorpConnectorConfigController()
	return fctl.NewCommand(internal.MoneycorpConnector+" <file>|-",
		fctl.WithShortDescription("Update the config of a Moneycorp connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithController[*UpdateMoneycorpConnectorConfigStore](c),
	)
}

func (c *UpdateMoneycorpConnectorConfigController) GetStore() *UpdateMoneycorpConnectorConfigStore {
	return c.store
}

func (c *UpdateMoneycorpConnectorConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("update configs are only supported in >= v1.0.0")
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

	config := &shared.MoneycorpConfig{}
	if err := json.Unmarshal([]byte(script), config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
		ConnectorConfig: shared.ConnectorConfig{
			MoneycorpConfig: config,
		},
		Connector:   shared.ConnectorMoneycorp,
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

func (c *UpdateMoneycorpConnectorConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' updated!", c.store.ConnectorID)

	return nil
}
