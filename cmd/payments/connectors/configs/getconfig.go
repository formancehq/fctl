package configs

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/collectionutils"

	"github.com/formancehq/fctl/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/cmd/payments/connectors/views"
	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type PaymentsLoadConfigStore struct {
	ConnectorConfig   *shared.ConnectorConfigResponse      `json:"connectorConfig"`
	V3ConnectorConfig *shared.V3GetConnectorConfigResponse `json:"v3ConnectorConfig,omitempty"`
	Provider          string                               `json:"provider"`
	ConnectorID       string                               `json:"connectorId"`
}

type PaymentsLoadConfigController struct {
	PaymentsVersion versions.Version

	store *PaymentsLoadConfigStore

	providerNameFlag string
	connectorIDFlag  string
}

func (c *PaymentsLoadConfigController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*PaymentsLoadConfigStore] = (*PaymentsLoadConfigController)(nil)

func NewDefaultPaymentsLoadConfigStore() *PaymentsLoadConfigStore {
	return &PaymentsLoadConfigStore{}
}

func NewPaymentsLoadConfigController() *PaymentsLoadConfigController {
	return &PaymentsLoadConfigController{
		store:            NewDefaultPaymentsLoadConfigStore(),
		providerNameFlag: "provider",
		connectorIDFlag:  "connector-id",
	}
}

func NewLoadConfigCommand() *cobra.Command {
	c := NewPaymentsLoadConfigController()
	return fctl.NewCommand("get-config",
		fctl.WithAliases("getconfig", "getconf", "gc", "get", "g"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag("provider", "", "Provider name (only used for v0, v1 or v2)"),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithShortDescription(fmt.Sprintf("Read a connector config (Connectors available: %v)", internal.AllConnectors)),
		fctl.WithController[*PaymentsLoadConfigStore](c),
	)
}

func (c *PaymentsLoadConfigController) GetStore() *PaymentsLoadConfigStore {
	return c.store
}

func (c *PaymentsLoadConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	provider := fctl.GetString(cmd, c.providerNameFlag)
	connectorID := fctl.GetString(cmd, c.connectorIDFlag)

	switch c.PaymentsVersion {
	case versions.V3:
		if connectorID == "" {
			return nil, fmt.Errorf("connector-id is required for v3")
		}

		response, err := stackClient.Payments.V3.GetConnectorConfig(cmd.Context(), operations.V3GetConnectorConfigRequest{
			ConnectorID: connectorID,
		})
		if err != nil {
			return nil, err
		}
		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		if response.V3GetConnectorConfigResponse == nil {
			return nil, fmt.Errorf("unexpected response: %v", response)
		}

		c.store.V3ConnectorConfig = response.V3GetConnectorConfigResponse
		c.store.ConnectorID = connectorID
		c.store.Provider = strings.ToLower(string(response.V3GetConnectorConfigResponse.Data.Type))

	case versions.V0:
		if provider == "" {
			return nil, fmt.Errorf("provider is required")
		}

		response, err := stackClient.Payments.V1.ReadConnectorConfig(cmd.Context(), operations.ReadConnectorConfigRequest{
			Connector: shared.Connector(provider),
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.Provider = provider
		c.store.ConnectorConfig = response.ConnectorConfigResponse

	default:
		connectorList, err := stackClient.Payments.V1.ListAllConnectors(cmd.Context())
		if err != nil {
			return nil, err
		}
		if connectorList.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", connectorList.StatusCode)
		}

		connectorsFiltered := collectionutils.Filter(connectorList.ConnectorsResponse.Data, func(connector shared.ConnectorsResponseData) bool {
			if connectorID != "" {
				return connector.ConnectorID == connectorID
			}

			if provider != "" {
				return connector.Provider == shared.Connector(strings.ToUpper(provider))
			}

			return true
		})

		switch len(connectorsFiltered) {
		case 0:
			return nil, fmt.Errorf("no connectors found")
		case 1:
			provider = string(connectorsFiltered[0].Provider)
			connectorID = connectorsFiltered[0].ConnectorID
		default:
			options := make([]string, 0, len(connectorsFiltered))
			for _, connector := range connectorsFiltered {
				options = append(options, strings.Join([]string{"id:" + connector.ConnectorID, "provider:" + string(connector.Provider), "name:" + connector.Name, "enabled:" + fctl.BoolPointerToString(connector.Enabled)}, " "))
			}
			printer := pterm.DefaultInteractiveSelect.WithOptions(options)
			selectedOption, err := printer.Show("Please select a connector")
			if err != nil {
				return nil, err
			}
			connectorID = strings.Split(strings.Split(selectedOption, " ")[0], ":")[1]
			provider = strings.Split(strings.Split(selectedOption, " ")[1], ":")[1]
		}

		response, err := stackClient.Payments.V1.ReadConnectorConfigV1(cmd.Context(), operations.ReadConnectorConfigV1Request{
			Connector:   shared.Connector(provider),
			ConnectorID: connectorID,
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.Provider = strings.ToLower(provider)
		c.store.ConnectorID = connectorID
		c.store.ConnectorConfig = response.ConnectorConfigResponse
	}

	return c, nil

}

// TODO: This need to use the ui.NewListModel
func (c *PaymentsLoadConfigController) Render(cmd *cobra.Command, args []string) error {
	if c.PaymentsVersion == versions.V3 {
		return c.renderV3(cmd, args)
	}
	return c.renderV1V2(cmd, args)
}

func (c *PaymentsLoadConfigController) renderV3(cmd *cobra.Command, args []string) error {
	var err error
	provider := c.store.Provider
	switch provider {
	case internal.StripeConnector:
		err = views.DisplayStripeConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.ModulrConnector:
		err = views.DisplayModulrConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.BankingCircleConnector:
		err = views.DisplayBankingCircleConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.CurrencyCloudConnector:
		err = views.DisplayCurrencyCloudConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.WiseConnector:
		err = views.DisplayWiseConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.MangoPayConnector:
		err = views.DisplayMangopayConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.MoneycorpConnector:
		err = views.DisplayMoneycorpConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.AtlarConnector:
		err = views.DisplayAtlarConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.AdyenConnector:
		err = views.DisplayAdyenConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.QontoConnector:
		err = views.DisplayQontoConfigV3(cmd, c.store.V3ConnectorConfig)
	case internal.ColumnConnector:
		err = views.DisplayColumnConfigV3(cmd, c.store.V3ConnectorConfig)
	default:
		pterm.Error.WithWriter(cmd.OutOrStderr()).Printfln("Unknown provider.")
	}

	return err
}

func (c *PaymentsLoadConfigController) renderV1V2(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorConfig == nil {
		return fmt.Errorf("no connector config available")
	}
	var err error
	switch c.store.Provider {
	case internal.StripeConnector:
		err = views.DisplayStripeConfig(cmd, c.store.ConnectorConfig)
	case internal.ModulrConnector:
		err = views.DisplayModulrConfig(cmd, c.store.ConnectorConfig)
	case internal.BankingCircleConnector:
		err = views.DisplayBankingCircleConfig(cmd, c.store.ConnectorConfig)
	case internal.CurrencyCloudConnector:
		err = views.DisplayCurrencyCloudConfig(cmd, c.store.ConnectorConfig)
	case internal.WiseConnector:
		err = views.DisplayWiseConfig(cmd, c.store.ConnectorConfig)
	case internal.MangoPayConnector:
		err = views.DisplayMangopayConfig(cmd, c.store.ConnectorConfig)
	case internal.MoneycorpConnector:
		err = views.DisplayMoneycorpConfig(cmd, c.store.ConnectorConfig)
	case internal.AtlarConnector:
		err = views.DisplayAtlarConfig(cmd, c.store.ConnectorConfig)
	case internal.AdyenConnector:
		err = views.DisplayAdyenConfig(cmd, c.store.ConnectorConfig)
	default:
		pterm.Error.WithWriter(cmd.OutOrStderr()).Printfln("Unknown provider.")
	}

	return err
}
