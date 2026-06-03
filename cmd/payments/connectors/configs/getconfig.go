package configs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
	"github.com/formancehq/go-libs/v4/collectionutils"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/v3/cmd/payments/connectors/views"
	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type PaymentsLoadConfigStore struct {
	ConnectorConfig *payments.ConnectorConfigResponse `json:"connectorConfig"`
	V3ConfigData    map[string]interface{}            `json:"v3ConnectorConfig,omitempty"`
	Provider        string                            `json:"provider"`
	ConnectorID     string                            `json:"connectorId"`
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

func NewGetConfigCommand() *cobra.Command {
	c := NewPaymentsLoadConfigController()
	return fctl.NewCommand("get-config",
		fctl.WithAliases("getconfig", "getconf", "gc", "get", "g"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag("provider", "", "Provider name (only used for v0, v1 or v2)"),
		fctl.WithStringFlag("connector-id", "", "Connector ID"),
		fctl.WithShortDescription("Read a connector config"),
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

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	provider := fctl.GetString(cmd, c.providerNameFlag)
	connectorID := fctl.GetString(cmd, c.connectorIDFlag)

	switch c.PaymentsVersion.Major {
	case versions.V3:
		if connectorID == "" {
			return nil, fmt.Errorf("connector-id is required for v3")
		}

		// Use raw HTTP to avoid the SDK's discriminated union failing on unknown connectors.
		clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil, err
		}

		url := strings.TrimRight(clients.URI, "/") + "/api/payments/v3/connectors/" + connectorID + "/config"
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")

		httpResp, err := clients.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer httpResp.Body.Close()

		respBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}
		if httpResp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(respBody, &envelope); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		data, ok := envelope["data"]
		if !ok {
			return nil, fmt.Errorf("unexpected response shape: missing 'data' key")
		}
		configData, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected response shape: 'data' is not an object")
		}

		c.store.V3ConfigData = configData
		c.store.ConnectorID = connectorID
		if p, ok := configData["provider"].(string); ok {
			c.store.Provider = strings.ToLower(p)
		}

	case versions.V0:
		stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil, err
		}
		if provider == "" {
			return nil, fmt.Errorf("provider is required")
		}

		response, err := stackClient.Payments.V1.ReadConnectorConfig(cmd.Context(), operations.ReadConnectorConfigRequest{
			Connector: payments.Connector(provider),
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
		stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil, err
		}

		connectorList, err := stackClient.Payments.V1.ListAllConnectors(cmd.Context())
		if err != nil {
			return nil, err
		}
		if connectorList.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", connectorList.StatusCode)
		}

		connectorsFiltered := collectionutils.Filter(connectorList.ConnectorsResponse.Data, func(connector payments.ConnectorsResponseData) bool {
			if connectorID != "" {
				return connector.ConnectorID == connectorID
			}

			if provider != "" {
				return connector.Connector == payments.Connector(strings.ToUpper(provider))
			}

			return true
		})

		switch len(connectorsFiltered) {
		case 0:
			return nil, fmt.Errorf("no connectors found")
		case 1:
			provider = string(connectorsFiltered[0].Connector)
			connectorID = connectorsFiltered[0].ConnectorID
		default:
			options := make([]string, 0, len(connectorsFiltered))
			for _, connector := range connectorsFiltered {
				options = append(options, strings.Join([]string{"id:" + connector.ConnectorID, "provider:" + string(connector.Connector), "name:" + connector.Name, "enabled:" + fctl.BoolPointerToString(connector.Enabled)}, " "))
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
			Connector:   payments.Connector(provider),
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

func (c *PaymentsLoadConfigController) Render(cmd *cobra.Command, args []string) error {
	if c.PaymentsVersion.Major == versions.V3 {
		return c.renderV3(cmd, args)
	}
	return c.renderV1V2(cmd, args)
}

func (c *PaymentsLoadConfigController) renderV3(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{{"Field", "Value"}}

	keys := make([]string, 0, len(c.store.V3ConfigData))
	for k := range c.store.V3ConfigData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := fmt.Sprintf("%v", c.store.V3ConfigData[k])
		tableData = append(tableData, []string{pterm.LightCyan(k + ":"), val})
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}

// TODO: This need to use the ui.NewListModel
func (c *PaymentsLoadConfigController) renderV1V2(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorConfig == nil {
		return fmt.Errorf("no connector config available")
	}
	var err error
	provider := c.store.Provider
	switch provider {
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
		err = fmt.Errorf("unknown provider: %s", provider)
		pterm.Error.WithWriter(cmd.OutOrStderr()).Printfln("%s", err.Error())
	}

	return err
}
