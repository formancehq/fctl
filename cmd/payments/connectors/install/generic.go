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

type PaymentsConnectorsGenericStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}
type PaymentsConnectorsGenericController struct {
	store *PaymentsConnectorsGenericStore
}

func NewDefaultPaymentsConnectorsGenericStore() *PaymentsConnectorsGenericStore {
	return &PaymentsConnectorsGenericStore{
		Success:       false,
		ConnectorName: internal.GenericConnector,
	}
}
func NewPaymentsConnectorsGenericController() *PaymentsConnectorsGenericController {
	return &PaymentsConnectorsGenericController{
		store: NewDefaultPaymentsConnectorsGenericStore(),
	}
}

func NewGenericCommand() *cobra.Command {
	c := NewPaymentsConnectorsGenericController()
	return fctl.NewCommand(internal.GenericConnector+" <file>|-",
		fctl.WithShortDescription("Install a Generic connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*PaymentsConnectorsGenericStore](c),
	)
}

func (c *PaymentsConnectorsGenericController) GetStore() *PaymentsConnectorsGenericStore {
	return c.store
}

func (c *PaymentsConnectorsGenericController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", internal.GenericConnector) {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	var config shared.GenericConfig
	if err := json.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}

	request := operations.InstallConnectorRequest{
		Connector: shared.ConnectorGeneric,
		ConnectorConfig: shared.ConnectorConfig{
			GenericConfig: &config,
		},
	}
	response, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("installing connector: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = true
	c.store.ConnectorName = internal.GenericConnector

	if response.ConnectorResponse != nil {
		c.store.ConnectorID = response.ConnectorResponse.Data.ConnectorID
	}

	return c, nil
}

func (c *PaymentsConnectorsGenericController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed!", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed!", c.store.ConnectorName, c.store.ConnectorID)
	}

	return nil
}
