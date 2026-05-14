package ledger

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	ProductLedger                    capabilities.Product = "ledger"
	FeatureAddTransactionMetadata    capabilities.Feature = "addMetadataOnTransaction"
	FeatureCountTransactions         capabilities.Feature = "countTransactions"
	FeatureCreateTransaction         capabilities.Feature = "createTransaction"
	FeatureDeleteTransactionMetadata capabilities.Feature = "deleteTransactionMetadata"
	FeatureListTransactions          capabilities.Feature = "listTransactions"
	FeatureGetTransaction            capabilities.Feature = "getTransaction"
	FeatureRevertTransaction         capabilities.Feature = "revertTransaction"
)

type ListTransactionsInput struct {
	Ledger      string
	PageSize    int64
	Cursor      string
	Account     string
	Source      string
	Destination string
	Reference   string
	Metadata    map[string]string
	StartTime   *time.Time
	EndTime     *time.Time
}

type ListTransactionsOutput struct {
	APIVersion   capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Transactions []TransactionSummary    `json:"transactions" yaml:"transactions"`
	HasMore      bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize     int64                   `json:"pageSize" yaml:"pageSize"`
	Next         *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous     *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type CountTransactionsInput struct {
	Ledger      string
	Account     string
	Source      string
	Destination string
	Reference   string
}

type CountTransactionsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Count      int64                   `json:"count" yaml:"count"`
}

type SendTransactionInput struct {
	Ledger      string
	Source      string
	Destination string
	Amount      string
	Asset       string
	Reference   string
	Metadata    map[string]string
}

type SendTransactionOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Transaction TransactionSummary      `json:"transaction" yaml:"transaction"`
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

type RevertTransactionInput struct {
	Ledger          string
	TransactionID   string
	AtEffectiveDate bool
	Force           bool
}

type RevertTransactionOutput struct {
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

type CountTransactionsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CountTransactionsInput) (CountTransactionsOutput, error)
}

type CountTransactionsService struct {
	Handlers []CountTransactionsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type SendTransactionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, SendTransactionInput) (SendTransactionOutput, error)
}

type SendTransactionService struct {
	Handlers []SendTransactionHandler
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

type RevertTransactionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, RevertTransactionInput) (RevertTransactionOutput, error)
}

type RevertTransactionService struct {
	Handlers []RevertTransactionHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListTransactionsService) Run(ctx context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
	if input.Ledger == "" {
		return ListTransactionsOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListTransactionsHandler{}
	for _, handler := range s.Handlers {
		if (input.StartTime != nil || input.EndTime != nil) && handler.APIVersion != "v1" {
			continue
		}
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

func (s CountTransactionsService) Run(ctx context.Context, input CountTransactionsInput) (CountTransactionsOutput, error) {
	if input.Ledger == "" {
		return CountTransactionsOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CountTransactionsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CountTransactionsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CountTransactionsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return CountTransactionsOutput{}, err
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

func SDKCountTransactionsHandlers(sdk *formance.Formance) []CountTransactionsHandler {
	return []CountTransactionsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CountTransactionsInput) (CountTransactionsOutput, error) {
				response, err := sdk.Ledger.V1.CountTransactions(ctx, toV1CountTransactionsRequest(input))
				if err != nil {
					return CountTransactionsOutput{}, err
				}
				count, err := countHeader(response.Headers)
				if err != nil {
					return CountTransactionsOutput{}, err
				}
				return CountTransactionsOutput{Count: count}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input CountTransactionsInput) (CountTransactionsOutput, error) {
				response, err := sdk.Ledger.V2.CountTransactions(ctx, toV2CountTransactionsRequest(input))
				if err != nil {
					return CountTransactionsOutput{}, err
				}
				count, err := countHeader(response.Headers)
				if err != nil {
					return CountTransactionsOutput{}, err
				}
				return CountTransactionsOutput{Count: count}, nil
			},
		},
	}
}

func (s SendTransactionService) Run(ctx context.Context, input SendTransactionInput) (SendTransactionOutput, error) {
	if input.Ledger == "" {
		return SendTransactionOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Source == "" {
		return SendTransactionOutput{}, fmt.Errorf("source is required")
	}
	if input.Destination == "" {
		return SendTransactionOutput{}, fmt.Errorf("destination is required")
	}
	if input.Amount == "" {
		return SendTransactionOutput{}, fmt.Errorf("amount is required")
	}
	if input.Asset == "" {
		return SendTransactionOutput{}, fmt.Errorf("asset is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]SendTransactionHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return SendTransactionOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return SendTransactionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return SendTransactionOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKSendTransactionHandlers(sdk *formance.Formance) []SendTransactionHandler {
	return []SendTransactionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input SendTransactionInput) (SendTransactionOutput, error) {
				amount, err := parseAmount(input.Amount)
				if err != nil {
					return SendTransactionOutput{}, err
				}
				response, err := sdk.Ledger.V1.CreateTransaction(ctx, operations.CreateTransactionRequest{
					Ledger: input.Ledger,
					PostTransaction: shared.PostTransaction{
						Metadata: stringMapToAny(input.Metadata),
						Postings: []shared.Posting{{
							Amount:      amount,
							Asset:       input.Asset,
							Destination: input.Destination,
							Source:      input.Source,
						}},
						Reference: optionalString(input.Reference),
					},
				})
				if err != nil {
					return SendTransactionOutput{}, err
				}
				if response.TransactionsResponse == nil || len(response.TransactionsResponse.Data) == 0 {
					return SendTransactionOutput{}, fmt.Errorf("ledger v1 create transaction returned no data")
				}
				return SendTransactionOutput{Transaction: fromV1Transaction(response.TransactionsResponse.Data[0])}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input SendTransactionInput) (SendTransactionOutput, error) {
				amount, err := parseAmount(input.Amount)
				if err != nil {
					return SendTransactionOutput{}, err
				}
				response, err := sdk.Ledger.V2.CreateTransaction(ctx, operations.V2CreateTransactionRequest{
					Ledger: input.Ledger,
					V2PostTransaction: shared.V2PostTransaction{
						Metadata: input.Metadata,
						Postings: []shared.V2Posting{{
							Amount:      amount,
							Asset:       input.Asset,
							Destination: input.Destination,
							Source:      input.Source,
						}},
						Reference: optionalString(input.Reference),
					},
				})
				if err != nil {
					return SendTransactionOutput{}, err
				}
				if response.V2CreateTransactionResponse == nil {
					return SendTransactionOutput{}, fmt.Errorf("ledger v2 create transaction returned no data")
				}
				return SendTransactionOutput{Transaction: fromV2Transaction(response.V2CreateTransactionResponse.Data)}, nil
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

func (s RevertTransactionService) Run(ctx context.Context, input RevertTransactionInput) (RevertTransactionOutput, error) {
	if input.Ledger == "" {
		return RevertTransactionOutput{}, fmt.Errorf("ledger is required")
	}
	if input.TransactionID == "" {
		return RevertTransactionOutput{}, fmt.Errorf("transaction id is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]RevertTransactionHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return RevertTransactionOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return RevertTransactionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return RevertTransactionOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKRevertTransactionHandlers(sdk *formance.Formance) []RevertTransactionHandler {
	return []RevertTransactionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input RevertTransactionInput) (RevertTransactionOutput, error) {
				if input.AtEffectiveDate {
					return RevertTransactionOutput{}, fmt.Errorf("--at-effective-date requires ledger API v2+")
				}
				txid, ok := new(big.Int).SetString(input.TransactionID, 10)
				if !ok {
					return RevertTransactionOutput{}, fmt.Errorf("transaction id must be an integer")
				}
				response, err := sdk.Ledger.V1.RevertTransaction(ctx, operations.RevertTransactionRequest{
					Ledger:        input.Ledger,
					Txid:          txid,
					DisableChecks: pointer(input.Force),
				})
				if err != nil {
					return RevertTransactionOutput{}, err
				}
				if response.TransactionResponse == nil {
					return RevertTransactionOutput{}, fmt.Errorf("ledger v1 revert transaction returned no data")
				}
				return RevertTransactionOutput{Transaction: fromV1Transaction(response.TransactionResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input RevertTransactionInput) (RevertTransactionOutput, error) {
				txid, ok := new(big.Int).SetString(input.TransactionID, 10)
				if !ok {
					return RevertTransactionOutput{}, fmt.Errorf("transaction id must be an integer")
				}
				response, err := sdk.Ledger.V2.RevertTransaction(ctx, operations.V2RevertTransactionRequest{
					Ledger:                     input.Ledger,
					ID:                         txid,
					AtEffectiveDate:            pointer(input.AtEffectiveDate),
					Force:                      pointer(input.Force),
					V2RevertTransactionRequest: &shared.V2RevertTransactionRequest{},
				})
				if err != nil {
					return RevertTransactionOutput{}, err
				}
				if response.V2RevertTransactionResponse == nil {
					return RevertTransactionOutput{}, fmt.Errorf("ledger v2 revert transaction returned no data")
				}
				return RevertTransactionOutput{Transaction: fromV2Transaction(response.V2RevertTransactionResponse.Data)}, nil
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
	if len(input.Metadata) > 0 {
		request.Metadata = stringMapToAny(input.Metadata)
	}
	if input.StartTime != nil {
		request.StartTime = input.StartTime
	}
	if input.EndTime != nil {
		request.EndTime = input.EndTime
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
	for key, value := range input.Metadata {
		query["metadata["+key+"]"] = value
	}
	if len(query) > 0 {
		request.Query = query
	}
	return request
}

func toV1CountTransactionsRequest(input CountTransactionsInput) operations.CountTransactionsRequest {
	request := operations.CountTransactionsRequest{
		Ledger: input.Ledger,
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

func toV2CountTransactionsRequest(input CountTransactionsInput) operations.V2CountTransactionsRequest {
	request := operations.V2CountTransactionsRequest{
		Ledger: input.Ledger,
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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func parseAmount(value string) (*big.Int, error) {
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("amount must be an integer")
	}
	return amount, nil
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

func countHeader(headers map[string][]string) (int64, error) {
	values := headers["Count"]
	if len(values) == 0 || values[0] == "" {
		return 0, fmt.Errorf("count response missing Count header")
	}
	count, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid Count header %q: %w", values[0], err)
	}
	return count, nil
}
