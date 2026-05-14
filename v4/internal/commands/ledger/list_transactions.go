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
	FeatureGetTransaction   capabilities.Feature = "getTransaction"
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
	APIVersion   capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Transactions []TransactionSummary    `json:"transactions" yaml:"transactions"`
	HasMore      bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize     int64                   `json:"pageSize" yaml:"pageSize"`
	Next         *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous     *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type TransactionSummary struct {
	ID        string         `json:"id" yaml:"id"`
	Reference *string        `json:"reference,omitempty" yaml:"reference,omitempty"`
	Timestamp time.Time      `json:"timestamp" yaml:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type GetTransactionInput struct {
	Ledger        string
	TransactionID string
}

type GetTransactionOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Transaction TransactionSummary      `json:"transaction" yaml:"transaction"`
}

type ListTransactionsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListTransactionsInput) (ListTransactionsOutput, error)
}

type ListTransactionsService struct {
	Handlers []ListTransactionsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetTransactionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetTransactionInput) (GetTransactionOutput, error)
}

type GetTransactionService struct {
	Handlers []GetTransactionHandler
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

func (s GetTransactionService) Run(ctx context.Context, input GetTransactionInput) (GetTransactionOutput, error) {
	if input.Ledger == "" {
		return GetTransactionOutput{}, fmt.Errorf("ledger is required")
	}
	if input.TransactionID == "" {
		return GetTransactionOutput{}, fmt.Errorf("transaction id is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetTransactionHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetTransactionOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetTransactionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetTransactionOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKGetTransactionHandlers(sdk *formance.Formance) []GetTransactionHandler {
	return []GetTransactionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetTransactionInput) (GetTransactionOutput, error) {
				txid, ok := new(big.Int).SetString(input.TransactionID, 10)
				if !ok {
					return GetTransactionOutput{}, fmt.Errorf("transaction id must be an integer")
				}
				response, err := sdk.Ledger.V1.GetTransaction(ctx, operations.GetTransactionRequest{
					Ledger: input.Ledger,
					Txid:   txid,
				})
				if err != nil {
					return GetTransactionOutput{}, err
				}
				if response.TransactionResponse == nil {
					return GetTransactionOutput{}, fmt.Errorf("ledger v1 get transaction returned no data")
				}
				return GetTransactionOutput{Transaction: fromV1Transaction(response.TransactionResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetTransactionInput) (GetTransactionOutput, error) {
				txid, ok := new(big.Int).SetString(input.TransactionID, 10)
				if !ok {
					return GetTransactionOutput{}, fmt.Errorf("transaction id must be an integer")
				}
				response, err := sdk.Ledger.V2.GetTransaction(ctx, operations.V2GetTransactionRequest{
					Ledger: input.Ledger,
					ID:     txid,
				})
				if err != nil {
					return GetTransactionOutput{}, err
				}
				if response.V2GetTransactionResponse == nil {
					return GetTransactionOutput{}, fmt.Errorf("ledger v2 get transaction returned no data")
				}
				return GetTransactionOutput{Transaction: fromV2Transaction(response.V2GetTransactionResponse.Data)}, nil
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
		transactions = append(transactions, fromV1Transaction(transaction))
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
		transactions = append(transactions, fromV2Transaction(transaction))
	}
	return ListTransactionsOutput{
		Transactions: transactions,
		HasMore:      cursor.HasMore,
		PageSize:     cursor.PageSize,
		Next:         cursor.Next,
		Previous:     cursor.Previous,
	}
}

func fromV1Transaction(transaction shared.Transaction) TransactionSummary {
	return TransactionSummary{
		ID:        bigIntString(transaction.Txid),
		Reference: transaction.Reference,
		Timestamp: transaction.Timestamp,
		Metadata:  transaction.Metadata,
	}
}

func fromV2Transaction(transaction shared.V2Transaction) TransactionSummary {
	return TransactionSummary{
		ID:        bigIntString(transaction.ID),
		Reference: transaction.Reference,
		Timestamp: transaction.Timestamp,
		Metadata:  stringMapToAny(transaction.Metadata),
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
