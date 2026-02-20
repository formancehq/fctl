package connectors

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

var (
	PaymentsConnectorsUninstall = "develop"
)

type PaymentsConnectorsUninstallStore struct {
	Success   bool   `json:"success"`
	Connector string `json:"connector"`

	// V3
	TaskID string `json:"task_id"`
}

type PaymentsConnectorsUninstallController struct {
	PaymentsVersion versions.Version

	store           *PaymentsConnectorsUninstallStore
	providerFlag    string
	connectorIDFlag string
}

func (c *PaymentsConnectorsUninstallController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*PaymentsConnectorsUninstallStore] = (*PaymentsConnectorsUninstallController)(nil)

func NewDefaultPaymentsConnectorsUninstallStore() *PaymentsConnectorsUninstallStore {
	return &PaymentsConnectorsUninstallStore{
		Success:   false,
		Connector: "",
		TaskID:    "",
	}
}

func NewPaymentsConnectorsUninstallController() *PaymentsConnectorsUninstallController {
	return &PaymentsConnectorsUninstallController{
		store:           NewDefaultPaymentsConnectorsUninstallStore(),
		providerFlag:    "provider",
		connectorIDFlag: "connector-id",
	}
}

func NewUninstallCommand() *cobra.Command {
	c := NewPaymentsConnectorsUninstallController()
	return fctl.NewCommand("uninstall",
		fctl.WithAliases("uninstall", "u", "un"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgs(internal.AllConnectors...),
		fctl.WithStringFlag(c.providerFlag, "", "Provider name"),
		fctl.WithStringFlag(c.connectorIDFlag, "", "Connector ID"),
		fctl.WithShortDescription("Uninstall a connector"),
		fctl.WithController[*PaymentsConnectorsUninstallStore](c),
	)
}

func (c *PaymentsConnectorsUninstallController) GetStore() *PaymentsConnectorsUninstallStore {
	return c.store
}

func (c *PaymentsConnectorsUninstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	provider := fctl.GetString(cmd, c.providerFlag)
	connectorID := fctl.GetString(cmd, c.connectorIDFlag)
	switch c.PaymentsVersion {
	case versions.V3:
		if connectorID == "" {
			return nil, fmt.Errorf("missing connector ID")
		}

		if !fctl.CheckStackApprobation(cmd, "You are about to uninstall connector '%s'", connectorID) {
			return nil, fctl.ErrMissingApproval
		}

		response, err := stackClient.Payments.V3.UninstallConnector(cmd.Context(), operations.V3UninstallConnectorRequest{
			ConnectorID: connectorID,
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.TaskID = response.V3UninstallConnectorResponse.Data.GetTaskID()

	case versions.V1:
		if provider == "" {
			return nil, fmt.Errorf("missing provider")
		}

		if connectorID == "" {
			return nil, fmt.Errorf("missing connector ID")
		}

		if !fctl.CheckStackApprobation(cmd, "You are about to uninstall connector '%s' from provider '%s'", connectorID, provider) {
			return nil, fctl.ErrMissingApproval
		}

		response, err := stackClient.Payments.V1.UninstallConnectorV1(cmd.Context(), operations.UninstallConnectorV1Request{
			ConnectorID: connectorID,
			Connector:   shared.Connector(provider),
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.Connector = connectorID
	case versions.V0:
		if provider == "" {
			return nil, fmt.Errorf("missing provider")
		}

		if !fctl.CheckStackApprobation(cmd, "You are about to uninstall connector '%s'", provider) {
			return nil, fctl.ErrMissingApproval
		}

		response, err := stackClient.Payments.V1.UninstallConnector(cmd.Context(), operations.UninstallConnectorRequest{
			Connector: shared.Connector(provider),
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.Connector = provider
	}

	c.store.Success = true

	return c, nil
}

// TODO: This need to use the ui.NewListModel
func (c *PaymentsConnectorsUninstallController) Render(cmd *cobra.Command, args []string) error {
	if c.PaymentsVersion < versions.V3 {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' uninstalled!", c.store.Connector)
		return nil
	}
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector uninstall scheduled with TaskID: %s", c.store.TaskID)
	return nil
}
