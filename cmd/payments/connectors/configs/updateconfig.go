package configs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConnectorUpdateConfigStore struct {
	Success     bool   `json:"success"`
	ConnectorID string `json:"connectorId"`
}

type ConnectorUpdateConfigController struct {
	PaymentsVersion versions.Version
	store           *ConnectorUpdateConfigStore
	connectorIDFlag string
}

func (c *ConnectorUpdateConfigController) SetVersion(v versions.Version) {
	c.PaymentsVersion = v
}

var _ fctl.Controller[*ConnectorUpdateConfigStore] = (*ConnectorUpdateConfigController)(nil)

func NewConnectorUpdateConfigController() *ConnectorUpdateConfigController {
	return &ConnectorUpdateConfigController{
		store:           &ConnectorUpdateConfigStore{},
		connectorIDFlag: "connector-id",
	}
}

func NewUpdateConfigCommand() *cobra.Command {
	c := NewConnectorUpdateConfigController()
	return fctl.NewCommand("update-config <connector> <file>|-",
		fctl.WithAliases("uc"),
		fctl.WithShortDescription("Update the config of a connector"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithConfirmFlag(),
		fctl.WithStringFlag("connector-id", "", "Connector ID (required for v3)"),
		fctl.WithController[*ConnectorUpdateConfigStore](c),
	)
}

func (c *ConnectorUpdateConfigController) GetStore() *ConnectorUpdateConfigStore {
	return c.store
}

func (c *ConnectorUpdateConfigController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	connectorName := args[0]

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	connectorID := fctl.GetString(cmd, c.connectorIDFlag)

	script, err := fctl.ReadFile(cmd, args[1])
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to update the config of connector '%s'", connectorName) {
		return nil, fctl.ErrMissingApproval
	}

	switch c.PaymentsVersion.Major {
	case versions.V3:
		if connectorID == "" {
			return nil, fmt.Errorf("--connector-id is required for payments API v3")
		}
		clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil, err
		}
		return c.runV3(cmd, clients, connectorName, connectorID, script)
	default:
		if connectorID == "" {
			return nil, fmt.Errorf("--connector-id is required")
		}
		return c.runV1Typed(cmd, connectorName, connectorID, script)
	}
}

func (c *ConnectorUpdateConfigController) runV3(
	cmd *cobra.Command,
	clients *fctl.StackClients,
	connectorName, connectorID, script string,
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

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(script), &configMap); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	configMap["provider"] = canonicalName
	body, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	url := strings.TrimRight(clients.URI, "/") + "/api/payments/v3/connectors/" + connectorID + "/config"
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpResp, err := clients.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updating connector config: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
	}

	c.store.Success = true
	c.store.ConnectorID = connectorID
	return c, nil
}

func (c *ConnectorUpdateConfigController) runV1Typed(
	cmd *cobra.Command,
	connectorName, connectorID, script string,
) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	sc, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	name := strings.ToLower(connectorName)
	switch name {
	case internal.AdyenConnector:
		config := &payments.AdyenConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{AdyenConfig: config},
			Connector:       payments.ConnectorAdyen,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.AtlarConnector:
		config := &payments.AtlarConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{AtlarConfig: config},
			Connector:       payments.ConnectorAtlar,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.BankingCircleConnector:
		config := &payments.BankingCircleConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{BankingCircleConfig: config},
			Connector:       payments.ConnectorBankingCircle,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.CurrencyCloudConnector:
		config := &payments.CurrencyCloudConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{CurrencyCloudConfig: config},
			Connector:       payments.ConnectorCurrencyCloud,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.MangoPayConnector:
		config := &payments.MangoPayConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{MangoPayConfig: config},
			Connector:       payments.ConnectorMangopay,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.ModulrConnector:
		config := &payments.ModulrConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{ModulrConfig: config},
			Connector:       payments.ConnectorModulr,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.MoneycorpConnector:
		config := &payments.MoneycorpConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{MoneycorpConfig: config},
			Connector:       payments.ConnectorMoneycorp,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.StripeConnector:
		config := &payments.StripeConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{StripeConfig: config},
			Connector:       payments.ConnectorStripe,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	case internal.WiseConnector:
		config := &payments.WiseConfig{}
		if err := json.Unmarshal([]byte(script), config); err != nil {
			return nil, err
		}
		resp, err := sc.Payments.V1.UpdateConnectorConfigV1(cmd.Context(), operations.UpdateConnectorConfigV1Request{
			ConnectorConfig: payments.ConnectorConfig{WiseConfig: config},
			Connector:       payments.ConnectorWise,
			ConnectorID:     connectorID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating connector config: %w", err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

	default:
		return nil, fmt.Errorf("connector %q is not supported on payments API v0/v1/v2; upgrade to v3 for dynamic connector support", connectorName)
	}

	c.store.Success = true
	c.store.ConnectorID = connectorID
	return c, nil
}

func (c *ConnectorUpdateConfigController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Connector '%s' config updated!", c.store.ConnectorID)
	return nil
}
