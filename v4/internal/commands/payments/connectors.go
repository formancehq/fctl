package payments

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	FeatureListConnectors     capabilities.Feature = "listConnectors"
	FeatureUninstallConnector capabilities.Feature = "uninstallConnector"
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

type ListConnectorsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListConnectorsInput) (ListConnectorsOutput, error)
}

type UninstallConnectorHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UninstallConnectorInput) (UninstallConnectorOutput, error)
}

type ListConnectorsService struct {
	Handlers []ListConnectorsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UninstallConnectorService struct {
	Handlers []UninstallConnectorHandler
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
