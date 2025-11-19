package install

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/connectors/internal"
	fctl "github.com/formancehq/fctl/pkg"
)

type PaymentsConnectorsAtlarStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}

type PaymentsConnectorsAtlarController struct {
	store *PaymentsConnectorsAtlarStore
}

var _ fctl.Controller[*PaymentsConnectorsAtlarStore] = (*PaymentsConnectorsAtlarController)(nil)

func NewDefaultPaymentsConnectorsAtlarStore() *PaymentsConnectorsAtlarStore {
	return &PaymentsConnectorsAtlarStore{
		Success: false,
	}
}

func NewPaymentsConnectorsAtlarController() *PaymentsConnectorsAtlarController {
	return &PaymentsConnectorsAtlarController{
		store: NewDefaultPaymentsConnectorsAtlarStore(),
	}
}

func NewAtlarCommand() *cobra.Command {
	c := NewPaymentsConnectorsAtlarController()
	return fctl.NewCommand(internal.AtlarConnector+" <file>|-",
		fctl.WithShortDescription("Install an atlar connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*PaymentsConnectorsAtlarStore](c),
	)
}

func (c *PaymentsConnectorsAtlarController) GetStore() *PaymentsConnectorsAtlarStore {
	return c.store
}

func (c *PaymentsConnectorsAtlarController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", internal.AtlarConnector) {
		return nil, fctl.ErrMissingApproval
	}
	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	var config shared.AtlarConfig
	if err := json.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
		ConnectorConfig: shared.ConnectorConfig{
			AtlarConfig: &config,
		},
		Connector: shared.ConnectorAtlar,
	})
	if err != nil {
		return nil, fmt.Errorf("installing connector: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = true
	c.store.ConnectorName = internal.AtlarConnector

	if response.ConnectorResponse != nil {
		c.store.ConnectorID = response.ConnectorResponse.Data.ConnectorID
	}

	return c, nil
}

func (c *PaymentsConnectorsAtlarController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed!", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed!", c.store.ConnectorName, c.store.ConnectorID)
	}

	return nil
}
