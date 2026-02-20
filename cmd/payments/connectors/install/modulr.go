package install

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

type PaymentsConnectorsModulrStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}

type PaymentsConnectorsModulrController struct {
	store *PaymentsConnectorsModulrStore
}

var _ fctl.Controller[*PaymentsConnectorsModulrStore] = (*PaymentsConnectorsModulrController)(nil)

func NewDefaultPaymentsConnectorsModulrStore() *PaymentsConnectorsModulrStore {
	return &PaymentsConnectorsModulrStore{
		Success: false,
	}
}

func NewPaymentsConnectorsModulrController() *PaymentsConnectorsModulrController {
	return &PaymentsConnectorsModulrController{
		store: NewDefaultPaymentsConnectorsModulrStore(),
	}
}

func NewModulrCommand() *cobra.Command {
	c := NewPaymentsConnectorsModulrController()
	return fctl.NewCommand(internal.ModulrConnector+" <file>|-",
		fctl.WithShortDescription("Install a Modulr connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*PaymentsConnectorsModulrStore](c),
	)
}

func (c *PaymentsConnectorsModulrController) GetStore() *PaymentsConnectorsModulrStore {
	return c.store
}

func (c *PaymentsConnectorsModulrController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", internal.ModulrConnector) {
		return nil, fctl.ErrMissingApproval
	}
	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	var config shared.ModulrConfig
	if err := json.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
		ConnectorConfig: shared.ConnectorConfig{
			ModulrConfig: &config,
		},
		Connector: shared.ConnectorModulr,
	})
	if err != nil {
		return nil, fmt.Errorf("installing connector: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = true
	c.store.ConnectorName = internal.ModulrConnector

	if response.ConnectorResponse != nil {
		c.store.ConnectorID = response.ConnectorResponse.Data.ConnectorID
	}

	return c, nil
}

func (c *PaymentsConnectorsModulrController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed!", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed!", c.store.ConnectorName, c.store.ConnectorID)
	}

	return nil
}
