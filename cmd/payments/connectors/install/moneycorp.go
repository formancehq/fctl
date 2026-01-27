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

type PaymentsConnectorsMoneycorpStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}
type PaymentsConnectorsMoneycorpController struct {
	store *PaymentsConnectorsMoneycorpStore
}

func NewDefaultPaymentsConnectorsMoneycorpStore() *PaymentsConnectorsMoneycorpStore {
	return &PaymentsConnectorsMoneycorpStore{
		Success:       false,
		ConnectorName: internal.MoneycorpConnector,
	}
}
func NewPaymentsConnectorsMoneycorpController() *PaymentsConnectorsMoneycorpController {
	return &PaymentsConnectorsMoneycorpController{
		store: NewDefaultPaymentsConnectorsMoneycorpStore(),
	}
}
func NewMoneycorpCommand() *cobra.Command {
	c := NewPaymentsConnectorsMoneycorpController()

	return fctl.NewCommand(internal.MoneycorpConnector+" <file>|-",
		fctl.WithShortDescription("Install a Moneycorp connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*PaymentsConnectorsMoneycorpStore](c),
	)
}

func (c *PaymentsConnectorsMoneycorpController) GetStore() *PaymentsConnectorsMoneycorpStore {
	return c.store
}

func (c *PaymentsConnectorsMoneycorpController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", internal.MoneycorpConnector) {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	var config shared.MoneycorpConfig
	if err := json.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}

	request := operations.InstallConnectorRequest{
		Connector: shared.ConnectorMoneycorp,
		ConnectorConfig: shared.ConnectorConfig{
			MoneycorpConfig: &config,
		},
	}
	response, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("installing connector: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector installed!")

	c.store.Success = true
	c.store.ConnectorName = internal.MoneycorpConnector

	if response.ConnectorResponse != nil {
		c.store.ConnectorID = response.ConnectorResponse.Data.ConnectorID
	}

	return c, nil
}

func (c *PaymentsConnectorsMoneycorpController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed!", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed!", c.store.ConnectorName, c.store.ConnectorID)
	}

	return nil
}
