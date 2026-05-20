package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConnectorInstallStore struct {
	Success       bool   `json:"success"`
	ConnectorName string `json:"connectorName"`
	ConnectorID   string `json:"connectorId"`
}

type ConnectorInstallController struct {
	PaymentsVersion versions.Version
	store           *ConnectorInstallStore
}

func (c *ConnectorInstallController) SetVersion(v versions.Version) {
	c.PaymentsVersion = v
}

var _ fctl.Controller[*ConnectorInstallStore] = (*ConnectorInstallController)(nil)

func NewConnectorInstallController() *ConnectorInstallController {
	return &ConnectorInstallController{
		store: &ConnectorInstallStore{},
	}
}

func (c *ConnectorInstallController) GetStore() *ConnectorInstallStore {
	return c.store
}

func (c *ConnectorInstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	connectorName := args[0]

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	script, err := fctl.ReadFile(cmd, args[1])
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to install connector '%s'", connectorName) {
		return nil, fctl.ErrMissingApproval
	}

	switch c.PaymentsVersion.Major {
	case versions.V3:
		clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil, err
		}
		return c.runV3(cmd, clients, connectorName, script)
	default:
		return c.runV1Typed(cmd, connectorName, script)
	}
}

func (c *ConnectorInstallController) runV3(
	cmd *cobra.Command,
	clients *fctl.StackClients,
	connectorName, script string,
) (fctl.Renderable, error) {
	// Resolve canonical provider name from the live configs endpoint.
	canonicalName := connectorName
	resp, err := clients.SDK.Payments.V3.ListConnectorConfigs(cmd.Context())
	if err == nil && resp.StatusCode < 300 && resp.V3ConnectorConfigsResponse != nil {
		for name := range resp.V3ConnectorConfigsResponse.Data {
			if strings.EqualFold(name, connectorName) {
				canonicalName = name
				break
			}
		}
	}

	// Inject/override the "provider" field in the config JSON.
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(script), &configMap); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	configMap["provider"] = canonicalName
	body, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	url := strings.TrimRight(clients.URI, "/") + "/api/payments/v3/connectors/install/" + strings.ToLower(canonicalName)
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpResp, err := clients.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("installing connector: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	c.store.Success = true
	c.store.ConnectorName = canonicalName
	c.store.ConnectorID = result.Data
	return c, nil
}

func (c *ConnectorInstallController) runV1Typed(
	cmd *cobra.Command,
	connectorName, script string,
) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	name := strings.ToLower(connectorName)
	switch name {
	case internal.AdyenConnector:
		var config shared.AdyenConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{AdyenConfig: &config},
			Connector:       shared.ConnectorAdyen,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.AtlarConnector:
		var config shared.AtlarConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{AtlarConfig: &config},
			Connector:       shared.ConnectorAtlar,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.BankingCircleConnector:
		var config shared.BankingCircleConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{BankingCircleConfig: &config},
			Connector:       shared.ConnectorBankingCircle,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.CurrencyCloudConnector:
		var config shared.CurrencyCloudConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{CurrencyCloudConfig: &config},
			Connector:       shared.ConnectorCurrencyCloud,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.MangoPayConnector:
		var config shared.MangoPayConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{MangoPayConfig: &config},
			Connector:       shared.ConnectorMangopay,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.ModulrConnector:
		var config shared.ModulrConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{ModulrConfig: &config},
			Connector:       shared.ConnectorModulr,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.MoneycorpConnector:
		var config shared.MoneycorpConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{MoneycorpConfig: &config},
			Connector:       shared.ConnectorMoneycorp,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.StripeConnector:
		var config shared.StripeConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{StripeConfig: &config},
			Connector:       shared.ConnectorStripe,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.WiseConnector:
		var config shared.WiseConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{WiseConfig: &config},
			Connector:       shared.ConnectorWise,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	case internal.GenericConnector:
		var config shared.GenericConfig
		if err := json.Unmarshal([]byte(script), &config); err != nil {
			return nil, err
		}
		resp, err := stackClient.Payments.V1.InstallConnector(cmd.Context(), operations.InstallConnectorRequest{
			ConnectorConfig: shared.ConnectorConfig{GenericConfig: &config},
			Connector:       shared.ConnectorGeneric,
		})
		if err != nil {
			return nil, fmt.Errorf("installing connector: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		c.store.Success = true
		c.store.ConnectorName = name
		if resp.ConnectorResponse != nil {
			c.store.ConnectorID = resp.ConnectorResponse.Data.ConnectorID
		}

	default:
		return nil, fmt.Errorf("connector %q is not supported on payments API v0/v1/v2; upgrade to v3 for dynamic connector support", connectorName)
	}

	return c, nil
}

func (c *ConnectorInstallController) Render(cmd *cobra.Command, args []string) error {
	if c.store.ConnectorID == "" {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector installed!", c.store.ConnectorName)
	} else {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("%s: connector '%s' installed!", c.store.ConnectorName, c.store.ConnectorID)
	}
	return nil
}
