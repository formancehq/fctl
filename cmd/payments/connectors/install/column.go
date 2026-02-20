package install

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type PaymentsConnectorsColumnStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}

type PaymentsConnectorsColumnController struct {
	PaymentsVersion versions.Version

	store *PaymentsConnectorsColumnStore
}

func (c *PaymentsConnectorsColumnController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*PaymentsConnectorsColumnStore] = (*PaymentsConnectorsColumnController)(nil)

func NewDefaultPaymentsConnectorsColumnStore() *PaymentsConnectorsColumnStore {
	return &PaymentsConnectorsColumnStore{}
}

func NewPaymentsConnectorsColumnController() *PaymentsConnectorsColumnController {
	return &PaymentsConnectorsColumnController{
		store: NewDefaultPaymentsConnectorsColumnStore(),
	}
}

func NewColumnCommand() *cobra.Command {
	c := NewPaymentsConnectorsColumnController()
	return fctl.NewCommand(internal.ColumnConnector+" <file>|-",
		fctl.WithShortDescription("Install a column connector"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*PaymentsConnectorsColumnStore](c),
	)
}

func (c *PaymentsConnectorsColumnController) GetStore() *PaymentsConnectorsColumnStore {
	return c.store
}

func (c *PaymentsConnectorsColumnController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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
	if c.PaymentsVersion < versions.V3 {
		return nil, fmt.Errorf("column connector is only supported in version >= v3.0.0")
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	var config shared.V3ColumnConfig
	if err := json.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", internal.ColumnConnector) {
		return nil, fctl.ErrMissingApproval
	}
	response, err := stackClient.Payments.V3.InstallConnector(cmd.Context(), operations.V3InstallConnectorRequest{
		V3InstallConnectorRequest: &shared.V3InstallConnectorRequest{
			V3ColumnConfig: &config,
			Type:           internal.ColumnConnector,
		},
		Connector: internal.ColumnConnector,
	})
	if err != nil {
		return nil, fmt.Errorf("unexpected error during installation: %w", err)
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = true
	c.store.ConnectorName = internal.ColumnConnector

	if response.V3InstallConnectorResponse != nil {
		c.store.ConnectorID = response.V3InstallConnectorResponse.GetData()
	}

	return c, nil
}

func (c *PaymentsConnectorsColumnController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed", c.store.ConnectorName, c.store.ConnectorID)
	}
	return nil
}
