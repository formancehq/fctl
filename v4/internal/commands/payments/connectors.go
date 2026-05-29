package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	FeatureListConnectors        capabilities.Feature = "listConnectors"
	FeatureUninstallConnector    capabilities.Feature = "uninstallConnector"
	FeatureInstallConnector      capabilities.Feature = "installConnector"
	FeatureGetConnectorConfig    capabilities.Feature = "getConnectorConfig"
	FeatureUpdateConnectorConfig capabilities.Feature = "updateConnectorConfig"
)

type ListConnectorsInput struct {
	PageSize int64
	Cursor   string
}

type ConnectorSummary struct {
	ID                   string    `json:"id" yaml:"id"`
	Name                 string    `json:"name" yaml:"name"`
	Provider             string    `json:"provider" yaml:"provider"`
	Reference            string    `json:"reference,omitempty" yaml:"reference,omitempty"`
	CreatedAt            time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	ScheduledForDeletion bool      `json:"scheduledForDeletion,omitempty" yaml:"scheduledForDeletion,omitempty"`
}

type ListConnectorsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Connectors []ConnectorSummary      `json:"connectors" yaml:"connectors"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type UninstallConnectorInput struct {
	ConnectorID string
	Provider    string
}

type UninstallConnectorOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ConnectorID string                  `json:"connectorID" yaml:"connectorID"`
	TaskID      string                  `json:"taskID,omitempty" yaml:"taskID,omitempty"`
}

type InstallConnectorInput struct {
	Connector string
	Config    []byte
}

type InstallConnectorOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Connector   string                  `json:"connector" yaml:"connector"`
	ConnectorID string                  `json:"connectorID,omitempty" yaml:"connectorID,omitempty"`
}

type GetConnectorConfigInput struct {
	ConnectorID string
	Provider    string
}

type GetConnectorConfigOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ConnectorID string                  `json:"connectorID" yaml:"connectorID"`
	Provider    string                  `json:"provider,omitempty" yaml:"provider,omitempty"`
	Config      json.RawMessage         `json:"config" yaml:"config"`
}

type UpdateConnectorConfigInput struct {
	ConnectorID string
	Provider    string
	Config      []byte
}

type UpdateConnectorConfigOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ConnectorID string                  `json:"connectorID" yaml:"connectorID"`
}

type ListConnectorsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListConnectorsInput) (ListConnectorsOutput, error)
}

type UninstallConnectorHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UninstallConnectorInput) (UninstallConnectorOutput, error)
}

type InstallConnectorHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, InstallConnectorInput) (InstallConnectorOutput, error)
}

type GetConnectorConfigHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetConnectorConfigInput) (GetConnectorConfigOutput, error)
}

type UpdateConnectorConfigHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UpdateConnectorConfigInput) (UpdateConnectorConfigOutput, error)
}

type ListConnectorsService struct {
	Handlers []ListConnectorsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UninstallConnectorService struct {
	Handlers []UninstallConnectorHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type InstallConnectorService struct {
	Handlers []InstallConnectorHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetConnectorConfigService struct {
	Handlers []GetConnectorConfigHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UpdateConnectorConfigService struct {
	Handlers []UpdateConnectorConfigHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListConnectorsService) Run(ctx context.Context, input ListConnectorsInput) (ListConnectorsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListConnectorsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListConnectorsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListConnectorsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListConnectorsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s UninstallConnectorService) Run(ctx context.Context, input UninstallConnectorInput) (UninstallConnectorOutput, error) {
	if input.ConnectorID == "" {
		return UninstallConnectorOutput{}, fmt.Errorf("connector id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]UninstallConnectorHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return UninstallConnectorOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return UninstallConnectorOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return UninstallConnectorOutput{}, err
	}
	output.APIVersion = selected
	if output.ConnectorID == "" {
		output.ConnectorID = input.ConnectorID
	}
	return output, nil
}

func (s InstallConnectorService) Run(ctx context.Context, input InstallConnectorInput) (InstallConnectorOutput, error) {
	if input.Connector == "" {
		return InstallConnectorOutput{}, fmt.Errorf("connector is required")
	}
	if len(input.Config) == 0 {
		return InstallConnectorOutput{}, fmt.Errorf("connector config is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]InstallConnectorHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return InstallConnectorOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return InstallConnectorOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return InstallConnectorOutput{}, err
	}
	output.APIVersion = selected
	if output.Connector == "" {
		output.Connector = input.Connector
	}
	return output, nil
}

func (s GetConnectorConfigService) Run(ctx context.Context, input GetConnectorConfigInput) (GetConnectorConfigOutput, error) {
	if input.ConnectorID == "" {
		return GetConnectorConfigOutput{}, fmt.Errorf("connector id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetConnectorConfigHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetConnectorConfigOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetConnectorConfigOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetConnectorConfigOutput{}, err
	}
	output.APIVersion = selected
	if output.ConnectorID == "" {
		output.ConnectorID = input.ConnectorID
	}
	return output, nil
}

func (s UpdateConnectorConfigService) Run(ctx context.Context, input UpdateConnectorConfigInput) (UpdateConnectorConfigOutput, error) {
	if input.ConnectorID == "" {
		return UpdateConnectorConfigOutput{}, fmt.Errorf("connector id is required")
	}
	if len(input.Config) == 0 {
		return UpdateConnectorConfigOutput{}, fmt.Errorf("connector config is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]UpdateConnectorConfigHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return UpdateConnectorConfigOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return UpdateConnectorConfigOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return UpdateConnectorConfigOutput{}, err
	}
	output.APIVersion = selected
	if output.ConnectorID == "" {
		output.ConnectorID = input.ConnectorID
	}
	return output, nil
}

func SDKListConnectorsHandlers(sdk *formance.Formance) []ListConnectorsHandler {
	return []ListConnectorsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListConnectorsInput) (ListConnectorsOutput, error) {
				response, err := sdk.Payments.V1.ListAllConnectors(ctx)
				if err != nil {
					return ListConnectorsOutput{}, err
				}
				if response.ConnectorsResponse == nil {
					return ListConnectorsOutput{}, fmt.Errorf("payments v1 list connectors returned no data")
				}
				return fromV1Connectors(response.ConnectorsResponse.Data, input.PageSize), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListConnectorsInput) (ListConnectorsOutput, error) {
				response, err := sdk.Payments.V3.ListConnectors(ctx, toV3ListConnectorsRequest(input))
				if err != nil {
					return ListConnectorsOutput{}, err
				}
				if response.V3ConnectorsCursorResponse == nil {
					return ListConnectorsOutput{}, fmt.Errorf("payments v3 list connectors returned no cursor")
				}
				return fromV3ConnectorsCursor(response.V3ConnectorsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKUninstallConnectorHandlers(sdk *formance.Formance) []UninstallConnectorHandler {
	return []UninstallConnectorHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input UninstallConnectorInput) (UninstallConnectorOutput, error) {
				if input.Provider == "" {
					return UninstallConnectorOutput{}, fmt.Errorf("provider is required when uninstalling a connector through payments API v1")
				}
				_, err := sdk.Payments.V1.UninstallConnectorV1(ctx, operations.UninstallConnectorV1Request{
					Connector:   shared.Connector(input.Provider),
					ConnectorID: input.ConnectorID,
				})
				if err != nil {
					return UninstallConnectorOutput{}, err
				}
				return UninstallConnectorOutput{ConnectorID: input.ConnectorID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input UninstallConnectorInput) (UninstallConnectorOutput, error) {
				response, err := sdk.Payments.V3.UninstallConnector(ctx, operations.V3UninstallConnectorRequest{
					ConnectorID: input.ConnectorID,
				})
				if err != nil {
					return UninstallConnectorOutput{}, err
				}
				if response.V3UninstallConnectorResponse == nil {
					return UninstallConnectorOutput{}, fmt.Errorf("payments v3 uninstall connector returned no data")
				}
				return UninstallConnectorOutput{
					ConnectorID: input.ConnectorID,
					TaskID:      response.V3UninstallConnectorResponse.Data.TaskID,
				}, nil
			},
		},
	}
}

func SDKInstallConnectorHandlers(sdk *formance.Formance) []InstallConnectorHandler {
	return []InstallConnectorHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input InstallConnectorInput) (InstallConnectorOutput, error) {
				config, err := parseV1ConnectorConfig(input.Connector, input.Config)
				if err != nil {
					return InstallConnectorOutput{}, err
				}
				response, err := sdk.Payments.V1.InstallConnector(ctx, operations.InstallConnectorRequest{
					Connector:       shared.Connector(toV1ConnectorPath(input.Connector)),
					ConnectorConfig: config,
				})
				if err != nil {
					return InstallConnectorOutput{}, err
				}
				output := InstallConnectorOutput{Connector: input.Connector}
				if response.ConnectorResponse != nil {
					output.ConnectorID = response.ConnectorResponse.Data.ConnectorID
				}
				return output, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input InstallConnectorInput) (InstallConnectorOutput, error) {
				config, err := parseV3ConnectorConfig(input.Connector, input.Config)
				if err != nil {
					return InstallConnectorOutput{}, err
				}
				response, err := sdk.Payments.V3.InstallConnector(ctx, operations.V3InstallConnectorRequest{
					Connector:                 normalizeConnectorName(input.Connector),
					V3InstallConnectorRequest: &config,
				})
				if err != nil {
					return InstallConnectorOutput{}, err
				}
				if response.V3InstallConnectorResponse == nil {
					return InstallConnectorOutput{}, fmt.Errorf("payments v3 install connector returned no data")
				}
				return InstallConnectorOutput{
					Connector:   input.Connector,
					ConnectorID: response.V3InstallConnectorResponse.Data,
				}, nil
			},
		},
	}
}

func SDKGetConnectorConfigHandlers(sdk *formance.Formance) []GetConnectorConfigHandler {
	return []GetConnectorConfigHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetConnectorConfigInput) (GetConnectorConfigOutput, error) {
				if input.Provider == "" {
					return GetConnectorConfigOutput{}, fmt.Errorf("provider is required when reading a connector config through payments API v1")
				}
				response, err := sdk.Payments.V1.ReadConnectorConfigV1(ctx, operations.ReadConnectorConfigV1Request{
					Connector:   shared.Connector(toV1ConnectorPath(input.Provider)),
					ConnectorID: input.ConnectorID,
				})
				if err != nil {
					return GetConnectorConfigOutput{}, err
				}
				if response.ConnectorConfigResponse == nil {
					return GetConnectorConfigOutput{}, fmt.Errorf("payments v1 get connector config returned no data")
				}
				config, err := json.Marshal(response.ConnectorConfigResponse.Data)
				if err != nil {
					return GetConnectorConfigOutput{}, err
				}
				return GetConnectorConfigOutput{
					ConnectorID: input.ConnectorID,
					Provider:    strings.ToLower(input.Provider),
					Config:      config,
				}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetConnectorConfigInput) (GetConnectorConfigOutput, error) {
				response, err := sdk.Payments.V3.GetConnectorConfig(ctx, operations.V3GetConnectorConfigRequest{
					ConnectorID: input.ConnectorID,
				})
				if err != nil {
					return GetConnectorConfigOutput{}, err
				}
				if response.V3GetConnectorConfigResponse == nil {
					return GetConnectorConfigOutput{}, fmt.Errorf("payments v3 get connector config returned no data")
				}
				config, err := json.Marshal(response.V3GetConnectorConfigResponse.Data)
				if err != nil {
					return GetConnectorConfigOutput{}, err
				}
				return GetConnectorConfigOutput{
					ConnectorID: input.ConnectorID,
					Provider:    strings.ToLower(string(response.V3GetConnectorConfigResponse.Data.Type)),
					Config:      config,
				}, nil
			},
		},
	}
}

func SDKUpdateConnectorConfigHandlers(sdk *formance.Formance) []UpdateConnectorConfigHandler {
	return []UpdateConnectorConfigHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input UpdateConnectorConfigInput) (UpdateConnectorConfigOutput, error) {
				if input.Provider == "" {
					return UpdateConnectorConfigOutput{}, fmt.Errorf("provider is required when updating a connector config through payments API v1")
				}
				config, err := parseV1ConnectorConfig(input.Provider, input.Config)
				if err != nil {
					return UpdateConnectorConfigOutput{}, err
				}
				_, err = sdk.Payments.V1.UpdateConnectorConfigV1(ctx, operations.UpdateConnectorConfigV1Request{
					Connector:       shared.Connector(toV1ConnectorPath(input.Provider)),
					ConnectorID:     input.ConnectorID,
					ConnectorConfig: config,
				})
				if err != nil {
					return UpdateConnectorConfigOutput{}, err
				}
				return UpdateConnectorConfigOutput{ConnectorID: input.ConnectorID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input UpdateConnectorConfigInput) (UpdateConnectorConfigOutput, error) {
				config, err := parseV3ConnectorConfig(input.Provider, input.Config)
				if err != nil {
					return UpdateConnectorConfigOutput{}, err
				}
				_, err = sdk.Payments.V3.V3UpdateConnectorConfig(ctx, operations.V3UpdateConnectorConfigRequest{
					ConnectorID:               input.ConnectorID,
					V3InstallConnectorRequest: &config,
				})
				if err != nil {
					return UpdateConnectorConfigOutput{}, err
				}
				return UpdateConnectorConfigOutput{ConnectorID: input.ConnectorID}, nil
			},
		},
	}
}

func toV3ListConnectorsRequest(input ListConnectorsInput) operations.V3ListConnectorsRequest {
	request := operations.V3ListConnectorsRequest{
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	return request
}

func fromV1Connectors(data []shared.ConnectorsResponseData, pageSize int64) ListConnectorsOutput {
	if pageSize > 0 && int64(len(data)) > pageSize {
		data = data[:pageSize]
	}
	connectors := make([]ConnectorSummary, 0, len(data))
	for _, connector := range data {
		connectors = append(connectors, ConnectorSummary{
			ID:       connector.ConnectorID,
			Name:     connector.Name,
			Provider: string(connector.Provider),
		})
	}
	return ListConnectorsOutput{
		Connectors: connectors,
		PageSize:   pageSize,
	}
}

func fromV3ConnectorsCursor(cursor shared.V3ConnectorsCursorResponseCursor) ListConnectorsOutput {
	connectors := make([]ConnectorSummary, 0, len(cursor.Data))
	for _, connector := range cursor.Data {
		connectors = append(connectors, ConnectorSummary{
			ID:                   connector.ID,
			Name:                 connector.Name,
			Provider:             connector.Provider,
			Reference:            connector.Reference,
			CreatedAt:            connector.CreatedAt,
			ScheduledForDeletion: connector.ScheduledForDeletion,
		})
	}
	return ListConnectorsOutput{
		Connectors: connectors,
		HasMore:    cursor.HasMore,
		PageSize:   cursor.PageSize,
		Next:       cursor.Next,
		Previous:   cursor.Previous,
	}
}

func parseV1ConnectorConfig(connector string, data []byte) (shared.ConnectorConfig, error) {
	data, err := ensureConnectorProvider(data, connector)
	if err != nil {
		return shared.ConnectorConfig{}, err
	}
	var config shared.ConnectorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return shared.ConnectorConfig{}, err
	}
	return config, nil
}

func parseV3ConnectorConfig(connector string, data []byte) (shared.V3InstallConnectorRequest, error) {
	data, err := ensureConnectorProvider(data, connector)
	if err != nil {
		return shared.V3InstallConnectorRequest{}, err
	}
	var config shared.V3InstallConnectorRequest
	if err := json.Unmarshal(data, &config); err != nil {
		return shared.V3InstallConnectorRequest{}, err
	}
	return config, nil
}

func ensureConnectorProvider(data []byte, connector string) ([]byte, error) {
	if connector == "" {
		return data, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if _, ok := payload["provider"]; !ok {
		payload["provider"] = connectorProviderDiscriminator(connector)
	}
	return json.Marshal(payload)
}

func connectorProviderDiscriminator(connector string) string {
	switch normalizeConnectorName(connector) {
	case "bankingcircle", "banking-circle":
		return "Bankingcircle"
	case "coinbaseprime", "coinbase-prime":
		return "Coinbaseprime"
	case "currencycloud", "currency-cloud":
		return "Currencycloud"
	case "dummy-pay", "dummypay":
		return "Dummypay"
	case "fireblocks":
		return "Fireblocks"
	case "generic":
		return "Generic"
	case "mangopay", "mango-pay":
		return "Mangopay"
	case "moneycorp":
		return "Moneycorp"
	default:
		normalized := normalizeConnectorName(connector)
		if normalized == "" {
			return ""
		}
		return strings.ToUpper(normalized[:1]) + normalized[1:]
	}
}

func normalizeConnectorName(connector string) string {
	return strings.ToLower(strings.TrimSpace(connector))
}

func toV1ConnectorPath(connector string) string {
	switch normalizeConnectorName(connector) {
	case "bankingcircle", "banking-circle":
		return string(shared.ConnectorBankingCircle)
	case "currencycloud", "currency-cloud":
		return string(shared.ConnectorCurrencyCloud)
	case "dummy-pay", "dummypay":
		return string(shared.ConnectorDummyPay)
	default:
		return strings.ToUpper(normalizeConnectorName(connector))
	}
}
