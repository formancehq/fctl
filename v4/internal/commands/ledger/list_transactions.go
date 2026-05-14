package ledger

import (
	"context"
	"fmt"
	"math/big"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	ProductLedger           capabilities.Product = "ledger"
	FeatureListTransactions capabilities.Feature = "listTransactions"
)

type ListTransactionsInput struct {
	Ledger      string
	PageSize    int64
	Cursor      string
	Account     string
	Source      string
	Destination string
	Reference   string
}

type ListTransactionsOutput struct {
	APIVersion   capabilities.APIVersion `json:"apiVersion"`
	Transactions []TransactionSummary    `json:"transactions"`
	HasMore      bool                    `json:"hasMore"`
	PageSize     int64                   `json:"pageSize"`
	Next         *string                 `json:"next,omitempty"`
	Previous     *string                 `json:"previous,omitempty"`
}

type TransactionSummary struct {
	ID        string         `json:"id"`
	Reference *string        `json:"reference,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ListTransactionsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListTransactionsInput) (ListTransactionsOutput, error)
}

type ListTransactionsService struct {
	Handlers []ListTransactionsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListTransactionsService) Run(ctx context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
	if input.Ledger == "" {
		return ListTransactionsOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListTransactionsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListTransactionsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListTransactionsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListTransactionsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListTransactionsHandlers(sdk *formance.Formance) []ListTransactionsHandler {
	return []ListTransactionsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
				response, err := sdk.Ledger.V1.ListTransactions(ctx, toV1ListTransactionsRequest(input))
				if err != nil {
					return ListTransactionsOutput{}, err
				}
				if response.TransactionsCursorResponse == nil {
					return ListTransactionsOutput{}, fmt.Errorf("ledger v1 list transactions returned no cursor")
				}
				return fromV1ListTransactions(response.TransactionsCursorResponse.Cursor), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
				response, err := sdk.Ledger.V2.ListTransactions(ctx, toV2ListTransactionsRequest(input))
				if err != nil {
					return ListTransactionsOutput{}, err
				}
				if response.V2TransactionsCursorResponse == nil {
					return ListTransactionsOutput{}, fmt.Errorf("ledger v2 list transactions returned no cursor")
				}
				return fromV2ListTransactions(response.V2TransactionsCursorResponse.Cursor), nil
			},
		},
	}
}

func toV1ListTransactionsRequest(input ListTransactionsInput) operations.ListTransactionsRequest {
	request := operations.ListTransactionsRequest{
		Ledger:   input.Ledger,
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	if input.Account != "" {
		request.Account = pointer(input.Account)
	}
	if input.Source != "" {
		request.Source = pointer(input.Source)
	}
	if input.Destination != "" {
		request.Destination = pointer(input.Destination)
	}
	if input.Reference != "" {
		request.Reference = pointer(input.Reference)
	}
	return request
}

func toV2ListTransactionsRequest(input ListTransactionsInput) operations.V2ListTransactionsRequest {
	request := operations.V2ListTransactionsRequest{
		Ledger:   input.Ledger,
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	query := map[string]any{}
	if input.Account != "" {
		query["account"] = input.Account
	}
	if input.Source != "" {
		query["source"] = input.Source
	}
	if input.Destination != "" {
		query["destination"] = input.Destination
	}
	if input.Reference != "" {
		query["reference"] = input.Reference
	}
	if len(query) > 0 {
		request.Query = query
	}
	return request
}

func fromV1ListTransactions(cursor shared.TransactionsCursorResponseCursor) ListTransactionsOutput {
	transactions := make([]TransactionSummary, 0, len(cursor.Data))
	for _, transaction := range cursor.Data {
		transactions = append(transactions, TransactionSummary{
			ID:        bigIntString(transaction.Txid),
			Reference: transaction.Reference,
			Timestamp: transaction.Timestamp,
			Metadata:  transaction.Metadata,
		})
	}
	return ListTransactionsOutput{
		Transactions: transactions,
		HasMore:      cursor.HasMore,
		PageSize:     cursor.PageSize,
		Next:         cursor.Next,
		Previous:     cursor.Previous,
	}
}

func fromV2ListTransactions(cursor shared.V2TransactionsCursorResponseCursor) ListTransactionsOutput {
	transactions := make([]TransactionSummary, 0, len(cursor.Data))
	for _, transaction := range cursor.Data {
		transactions = append(transactions, TransactionSummary{
			ID:        bigIntString(transaction.ID),
			Reference: transaction.Reference,
			Timestamp: transaction.Timestamp,
			Metadata:  stringMapToAny(transaction.Metadata),
		})
	}
	return ListTransactionsOutput{
		Transactions: transactions,
		HasMore:      cursor.HasMore,
		PageSize:     cursor.PageSize,
		Next:         cursor.Next,
		Previous:     cursor.Previous,
	}
}

func pointer[T any](value T) *T {
	return &value
}

func bigIntString(value *big.Int) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func stringMapToAny(values map[string]string) map[string]any {
	if len(values) == 0 {
		return nil
	}
	ret := make(map[string]any, len(values))
	for key, value := range values {
		ret[key] = value
	}
	return ret
}
