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
	FeatureApprovePaymentInitiation capabilities.Feature = "approvePaymentInitiation"
	FeatureDeletePaymentInitiation  capabilities.Feature = "deletePaymentInitiation"
	FeatureGetTransferInitiation    capabilities.Feature = "getTransferInitiation"
	FeatureListTransferInitiation   capabilities.Feature = "listTransferInitiations"
	FeatureRejectPaymentInitiation  capabilities.Feature = "rejectPaymentInitiation"
	FeatureRetryPaymentInitiation   capabilities.Feature = "retryPaymentInitiation"
)

type TransferInitiationSummary struct {
	ID                   string            `json:"id" yaml:"id"`
	Reference            string            `json:"reference" yaml:"reference"`
	Type                 string            `json:"type" yaml:"type"`
	Status               string            `json:"status" yaml:"status"`
	Asset                string            `json:"asset" yaml:"asset"`
	Amount               string            `json:"amount" yaml:"amount"`
	InitialAmount        string            `json:"initialAmount,omitempty" yaml:"initialAmount,omitempty"`
	ConnectorID          string            `json:"connectorID" yaml:"connectorID"`
	Provider             string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	SourceAccountID      string            `json:"sourceAccountID,omitempty" yaml:"sourceAccountID,omitempty"`
	DestinationAccountID string            `json:"destinationAccountID,omitempty" yaml:"destinationAccountID,omitempty"`
	Description          string            `json:"description,omitempty" yaml:"description,omitempty"`
	Error                string            `json:"error,omitempty" yaml:"error,omitempty"`
	CreatedAt            time.Time         `json:"createdAt" yaml:"createdAt"`
	ScheduledAt          time.Time         `json:"scheduledAt" yaml:"scheduledAt"`
	Metadata             map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListTransferInitiationsInput struct {
	PageSize int64
	Cursor   string
	Query    string
}

type ListTransferInitiationsOutput struct {
	APIVersion          capabilities.APIVersion     `json:"apiVersion" yaml:"apiVersion"`
	TransferInitiations []TransferInitiationSummary `json:"transferInitiations" yaml:"transferInitiations"`
	HasMore             bool                        `json:"hasMore" yaml:"hasMore"`
	PageSize            int64                       `json:"pageSize" yaml:"pageSize"`
	Next                *string                     `json:"next,omitempty" yaml:"next,omitempty"`
	Previous            *string                     `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetTransferInitiationInput struct {
	TransferInitiationID string
}

type GetTransferInitiationOutput struct {
	APIVersion         capabilities.APIVersion   `json:"apiVersion" yaml:"apiVersion"`
	TransferInitiation TransferInitiationSummary `json:"transferInitiation" yaml:"transferInitiation"`
}

type TransferInitiationActionInput struct {
	TransferInitiationID string
}

type TransferInitiationActionOutput struct {
	APIVersion           capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	TransferInitiationID string                  `json:"transferInitiationID" yaml:"transferInitiationID"`
	TaskID               string                  `json:"taskID,omitempty" yaml:"taskID,omitempty"`
}

type ListTransferInitiationsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListTransferInitiationsInput) (ListTransferInitiationsOutput, error)
}

type GetTransferInitiationHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetTransferInitiationInput) (GetTransferInitiationOutput, error)
}

type TransferInitiationActionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, TransferInitiationActionInput) (TransferInitiationActionOutput, error)
}

type ListTransferInitiationsService struct {
	Handlers []ListTransferInitiationsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetTransferInitiationService struct {
	Handlers []GetTransferInitiationHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type TransferInitiationActionService struct {
	Handlers []TransferInitiationActionHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListTransferInitiationsService) Run(ctx context.Context, input ListTransferInitiationsInput) (ListTransferInitiationsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListTransferInitiationsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListTransferInitiationsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListTransferInitiationsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListTransferInitiationsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetTransferInitiationService) Run(ctx context.Context, input GetTransferInitiationInput) (GetTransferInitiationOutput, error) {
	if input.TransferInitiationID == "" {
		return GetTransferInitiationOutput{}, fmt.Errorf("transfer initiation id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetTransferInitiationHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetTransferInitiationOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetTransferInitiationOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetTransferInitiationOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s TransferInitiationActionService) Run(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
	if input.TransferInitiationID == "" {
		return TransferInitiationActionOutput{}, fmt.Errorf("transfer initiation id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]TransferInitiationActionHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return TransferInitiationActionOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return TransferInitiationActionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return TransferInitiationActionOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListTransferInitiationsHandlers(sdk *formance.Formance) []ListTransferInitiationsHandler {
	return []ListTransferInitiationsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListTransferInitiationsInput) (ListTransferInitiationsOutput, error) {
				response, err := sdk.Payments.V1.ListTransferInitiations(ctx, operations.ListTransferInitiationsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Query:    optionalString(input.Query),
				})
				if err != nil {
					return ListTransferInitiationsOutput{}, err
				}
				if response.TransferInitiationsCursor == nil {
					return ListTransferInitiationsOutput{}, fmt.Errorf("payments v1 list transfer initiations returned no cursor")
				}
				return fromV1TransferInitiationsCursor(response.TransferInitiationsCursor.Cursor), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListTransferInitiationsInput) (ListTransferInitiationsOutput, error) {
				response, err := sdk.Payments.V3.ListPaymentInitiations(ctx, operations.V3ListPaymentInitiationsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListTransferInitiationsOutput{}, err
				}
				if response.V3PaymentInitiationsCursorResponse == nil {
					return ListTransferInitiationsOutput{}, fmt.Errorf("payments v3 list payment initiations returned no cursor")
				}
				return fromV3PaymentInitiationsCursor(response.V3PaymentInitiationsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetTransferInitiationHandlers(sdk *formance.Formance) []GetTransferInitiationHandler {
	return []GetTransferInitiationHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetTransferInitiationInput) (GetTransferInitiationOutput, error) {
				response, err := sdk.Payments.V1.GetTransferInitiation(ctx, operations.GetTransferInitiationRequest{TransferID: input.TransferInitiationID})
				if err != nil {
					return GetTransferInitiationOutput{}, err
				}
				if response.TransferInitiationResponse == nil {
					return GetTransferInitiationOutput{}, fmt.Errorf("payments v1 get transfer initiation returned no data")
				}
				return GetTransferInitiationOutput{TransferInitiation: fromV1TransferInitiation(response.TransferInitiationResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetTransferInitiationInput) (GetTransferInitiationOutput, error) {
				response, err := sdk.Payments.V3.GetPaymentInitiation(ctx, operations.V3GetPaymentInitiationRequest{PaymentInitiationID: input.TransferInitiationID})
				if err != nil {
					return GetTransferInitiationOutput{}, err
				}
				if response.V3GetPaymentInitiationResponse == nil {
					return GetTransferInitiationOutput{}, fmt.Errorf("payments v3 get payment initiation returned no data")
				}
				return GetTransferInitiationOutput{TransferInitiation: fromV3PaymentInitiation(response.V3GetPaymentInitiationResponse.Data)}, nil
			},
		},
	}
}

func SDKApprovePaymentInitiationHandlers(sdk *formance.Formance) []TransferInitiationActionHandler {
	return []TransferInitiationActionHandler{
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				response, err := sdk.Payments.V3.ApprovePaymentInitiation(ctx, operations.V3ApprovePaymentInitiationRequest{
					PaymentInitiationID: input.TransferInitiationID,
				})
				if err != nil {
					return TransferInitiationActionOutput{}, err
				}
				if response.V3ApprovePaymentInitiationResponse == nil {
					return TransferInitiationActionOutput{}, fmt.Errorf("payments v3 approve payment initiation returned no data")
				}
				return TransferInitiationActionOutput{
					TransferInitiationID: input.TransferInitiationID,
					TaskID:               response.V3ApprovePaymentInitiationResponse.Data.TaskID,
				}, nil
			},
		},
	}
}

func SDKRejectPaymentInitiationHandlers(sdk *formance.Formance) []TransferInitiationActionHandler {
	return []TransferInitiationActionHandler{
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				if _, err := sdk.Payments.V3.RejectPaymentInitiation(ctx, operations.V3RejectPaymentInitiationRequest{
					PaymentInitiationID: input.TransferInitiationID,
				}); err != nil {
					return TransferInitiationActionOutput{}, err
				}
				return TransferInitiationActionOutput{TransferInitiationID: input.TransferInitiationID}, nil
			},
		},
	}
}

func SDKRetryPaymentInitiationHandlers(sdk *formance.Formance) []TransferInitiationActionHandler {
	return []TransferInitiationActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				if _, err := sdk.Payments.V1.RetryTransferInitiation(ctx, operations.RetryTransferInitiationRequest{
					TransferID: input.TransferInitiationID,
				}); err != nil {
					return TransferInitiationActionOutput{}, err
				}
				return TransferInitiationActionOutput{TransferInitiationID: input.TransferInitiationID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				response, err := sdk.Payments.V3.RetryPaymentInitiation(ctx, operations.V3RetryPaymentInitiationRequest{
					PaymentInitiationID: input.TransferInitiationID,
				})
				if err != nil {
					return TransferInitiationActionOutput{}, err
				}
				if response.V3RetryPaymentInitiationResponse == nil {
					return TransferInitiationActionOutput{}, fmt.Errorf("payments v3 retry payment initiation returned no data")
				}
				return TransferInitiationActionOutput{
					TransferInitiationID: input.TransferInitiationID,
					TaskID:               response.V3RetryPaymentInitiationResponse.Data.TaskID,
				}, nil
			},
		},
	}
}

func SDKDeletePaymentInitiationHandlers(sdk *formance.Formance) []TransferInitiationActionHandler {
	return []TransferInitiationActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				if _, err := sdk.Payments.V1.DeleteTransferInitiation(ctx, operations.DeleteTransferInitiationRequest{
					TransferID: input.TransferInitiationID,
				}); err != nil {
					return TransferInitiationActionOutput{}, err
				}
				return TransferInitiationActionOutput{TransferInitiationID: input.TransferInitiationID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
				if _, err := sdk.Payments.V3.DeletePaymentInitiation(ctx, operations.V3DeletePaymentInitiationRequest{
					PaymentInitiationID: input.TransferInitiationID,
				}); err != nil {
					return TransferInitiationActionOutput{}, err
				}
				return TransferInitiationActionOutput{TransferInitiationID: input.TransferInitiationID}, nil
			},
		},
	}
}

func fromV1TransferInitiationsCursor(cursor shared.TransferInitiationsCursorCursor) ListTransferInitiationsOutput {
	transfers := make([]TransferInitiationSummary, 0, len(cursor.Data))
	for _, transfer := range cursor.Data {
		transfers = append(transfers, fromV1TransferInitiation(transfer))
	}
	return ListTransferInitiationsOutput{TransferInitiations: transfers, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV3PaymentInitiationsCursor(cursor shared.V3PaymentInitiationsCursorResponseCursor) ListTransferInitiationsOutput {
	transfers := make([]TransferInitiationSummary, 0, len(cursor.Data))
	for _, transfer := range cursor.Data {
		transfers = append(transfers, fromV3PaymentInitiation(transfer))
	}
	return ListTransferInitiationsOutput{TransferInitiations: transfers, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV1TransferInitiation(transfer shared.TransferInitiation) TransferInitiationSummary {
	return TransferInitiationSummary{
		ID:                   transfer.ID,
		Reference:            transfer.Reference,
		Type:                 string(transfer.Type),
		Status:               string(transfer.Status),
		Asset:                transfer.Asset,
		Amount:               bigIntString(transfer.Amount),
		InitialAmount:        bigIntString(transfer.InitialAmount),
		ConnectorID:          transfer.ConnectorID,
		Provider:             stringValue(transfer.Provider),
		SourceAccountID:      transfer.SourceAccountID,
		DestinationAccountID: transfer.DestinationAccountID,
		Description:          transfer.Description,
		Error:                stringValue(transfer.Error),
		CreatedAt:            transfer.CreatedAt,
		ScheduledAt:          transfer.ScheduledAt,
		Metadata:             transfer.Metadata,
	}
}

func fromV3PaymentInitiation(transfer shared.V3PaymentInitiation) TransferInitiationSummary {
	return TransferInitiationSummary{
		ID:                   transfer.ID,
		Reference:            transfer.Reference,
		Type:                 string(transfer.Type),
		Status:               string(transfer.Status),
		Asset:                transfer.Asset,
		Amount:               bigIntString(transfer.Amount),
		ConnectorID:          transfer.ConnectorID,
		Provider:             transfer.Provider,
		SourceAccountID:      stringValue(transfer.SourceAccountID),
		DestinationAccountID: stringValue(transfer.DestinationAccountID),
		Description:          transfer.Description,
		Error:                stringValue(transfer.Error),
		CreatedAt:            transfer.CreatedAt,
		ScheduledAt:          transfer.ScheduledAt,
		Metadata:             transfer.Metadata,
	}
}
