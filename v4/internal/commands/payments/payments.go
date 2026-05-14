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
	FeatureGetPayment   capabilities.Feature = "getPayment"
	FeatureListPayments capabilities.Feature = "listPayments"
)

type PaymentSummary struct {
	ID                   string            `json:"id" yaml:"id"`
	Reference            string            `json:"reference" yaml:"reference"`
	Type                 string            `json:"type" yaml:"type"`
	Status               string            `json:"status" yaml:"status"`
	Scheme               string            `json:"scheme" yaml:"scheme"`
	Asset                string            `json:"asset" yaml:"asset"`
	Amount               string            `json:"amount" yaml:"amount"`
	InitialAmount        string            `json:"initialAmount" yaml:"initialAmount"`
	ConnectorID          string            `json:"connectorID" yaml:"connectorID"`
	SourceAccountID      string            `json:"sourceAccountID,omitempty" yaml:"sourceAccountID,omitempty"`
	DestinationAccountID string            `json:"destinationAccountID,omitempty" yaml:"destinationAccountID,omitempty"`
	CreatedAt            time.Time         `json:"createdAt" yaml:"createdAt"`
	Metadata             map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListPaymentsInput struct {
	PageSize int64
	Cursor   string
}

type ListPaymentsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Payments   []PaymentSummary        `json:"payments" yaml:"payments"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetPaymentInput struct {
	PaymentID string
}

type GetPaymentOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Payment    PaymentSummary          `json:"payment" yaml:"payment"`
}

type ListPaymentsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListPaymentsInput) (ListPaymentsOutput, error)
}

type GetPaymentHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetPaymentInput) (GetPaymentOutput, error)
}

type ListPaymentsService struct {
	Handlers []ListPaymentsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetPaymentService struct {
	Handlers []GetPaymentHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListPaymentsService) Run(ctx context.Context, input ListPaymentsInput) (ListPaymentsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListPaymentsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListPaymentsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListPaymentsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListPaymentsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetPaymentService) Run(ctx context.Context, input GetPaymentInput) (GetPaymentOutput, error) {
	if input.PaymentID == "" {
		return GetPaymentOutput{}, fmt.Errorf("payment id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetPaymentHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetPaymentOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetPaymentOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetPaymentOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListPaymentsHandlers(sdk *formance.Formance) []ListPaymentsHandler {
	return []ListPaymentsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListPaymentsInput) (ListPaymentsOutput, error) {
				response, err := sdk.Payments.V1.ListPayments(ctx, operations.ListPaymentsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListPaymentsOutput{}, err
				}
				if response.PaymentsCursor == nil {
					return ListPaymentsOutput{}, fmt.Errorf("payments v1 list payments returned no cursor")
				}
				return fromV1PaymentsCursor(response.PaymentsCursor.Cursor), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListPaymentsInput) (ListPaymentsOutput, error) {
				response, err := sdk.Payments.V3.ListPayments(ctx, operations.V3ListPaymentsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListPaymentsOutput{}, err
				}
				if response.V3PaymentsCursorResponse == nil {
					return ListPaymentsOutput{}, fmt.Errorf("payments v3 list payments returned no cursor")
				}
				return fromV3PaymentsCursor(response.V3PaymentsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetPaymentHandlers(sdk *formance.Formance) []GetPaymentHandler {
	return []GetPaymentHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetPaymentInput) (GetPaymentOutput, error) {
				response, err := sdk.Payments.V1.GetPayment(ctx, operations.GetPaymentRequest{PaymentID: input.PaymentID})
				if err != nil {
					return GetPaymentOutput{}, err
				}
				if response.PaymentResponse == nil {
					return GetPaymentOutput{}, fmt.Errorf("payments v1 get payment returned no data")
				}
				return GetPaymentOutput{Payment: fromV1Payment(response.PaymentResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetPaymentInput) (GetPaymentOutput, error) {
				response, err := sdk.Payments.V3.GetPayment(ctx, operations.V3GetPaymentRequest{PaymentID: input.PaymentID})
				if err != nil {
					return GetPaymentOutput{}, err
				}
				if response.V3GetPaymentResponse == nil {
					return GetPaymentOutput{}, fmt.Errorf("payments v3 get payment returned no data")
				}
				return GetPaymentOutput{Payment: fromV3Payment(response.V3GetPaymentResponse.Data)}, nil
			},
		},
	}
}

func fromV1PaymentsCursor(cursor shared.PaymentsCursorCursor) ListPaymentsOutput {
	payments := make([]PaymentSummary, 0, len(cursor.Data))
	for _, payment := range cursor.Data {
		payments = append(payments, fromV1Payment(payment))
	}
	return ListPaymentsOutput{Payments: payments, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV3PaymentsCursor(cursor shared.V3PaymentsCursorResponseCursor) ListPaymentsOutput {
	payments := make([]PaymentSummary, 0, len(cursor.Data))
	for _, payment := range cursor.Data {
		payments = append(payments, fromV3Payment(payment))
	}
	return ListPaymentsOutput{Payments: payments, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV1Payment(payment shared.Payment) PaymentSummary {
	return PaymentSummary{
		ID:                   payment.ID,
		Reference:            payment.Reference,
		Type:                 string(payment.Type),
		Status:               string(payment.Status),
		Scheme:               string(payment.Scheme),
		Asset:                payment.Asset,
		Amount:               bigIntString(payment.Amount),
		InitialAmount:        bigIntString(payment.InitialAmount),
		ConnectorID:          payment.ConnectorID,
		SourceAccountID:      payment.SourceAccountID,
		DestinationAccountID: payment.DestinationAccountID,
		CreatedAt:            payment.CreatedAt,
		Metadata:             payment.Metadata,
	}
}

func fromV3Payment(payment shared.V3Payment) PaymentSummary {
	return PaymentSummary{
		ID:                   payment.ID,
		Reference:            payment.Reference,
		Type:                 string(payment.Type),
		Status:               string(payment.Status),
		Scheme:               payment.Scheme,
		Asset:                payment.Asset,
		Amount:               bigIntString(payment.Amount),
		InitialAmount:        bigIntString(payment.InitialAmount),
		ConnectorID:          payment.ConnectorID,
		SourceAccountID:      stringValue(payment.SourceAccountID),
		DestinationAccountID: stringValue(payment.DestinationAccountID),
		CreatedAt:            payment.CreatedAt,
		Metadata:             payment.Metadata,
	}
}
