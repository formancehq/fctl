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
	ProductPayments     capabilities.Product = "payments"
	FeatureListAccounts capabilities.Feature = "listAccounts"
	FeatureGetAccount   capabilities.Feature = "getAccount"
)

type ListAccountsInput struct {
	PageSize int64
	Cursor   string
}

type AccountSummary struct {
	ID              string            `json:"id" yaml:"id"`
	Reference       string            `json:"reference" yaml:"reference"`
	Name            string            `json:"name" yaml:"name"`
	CreatedAt       time.Time         `json:"createdAt" yaml:"createdAt"`
	ConnectorID     string            `json:"connectorID" yaml:"connectorID"`
	DefaultAsset    string            `json:"defaultAsset,omitempty" yaml:"defaultAsset,omitempty"`
	DefaultCurrency string            `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	Provider        string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Type            string            `json:"type" yaml:"type"`
	Metadata        map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListAccountsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Accounts   []AccountSummary        `json:"accounts" yaml:"accounts"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetAccountInput struct {
	AccountID string
}

type GetAccountOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Account    AccountSummary          `json:"account" yaml:"account"`
}

type ListAccountsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListAccountsInput) (ListAccountsOutput, error)
}

type GetAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetAccountInput) (GetAccountOutput, error)
}

type ListAccountsService struct {
	Handlers []ListAccountsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetAccountService struct {
	Handlers []GetAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListAccountsService) Run(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListAccountsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListAccountsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListAccountsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListAccountsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetAccountService) Run(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
	if input.AccountID == "" {
		return GetAccountOutput{}, fmt.Errorf("account id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetAccountHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetAccountOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetAccountOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetAccountOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListAccountsHandlers(sdk *formance.Formance) []ListAccountsHandler {
	return []ListAccountsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
				response, err := sdk.Payments.V1.PaymentslistAccounts(ctx, toV1ListAccountsRequest(input))
				if err != nil {
					return ListAccountsOutput{}, err
				}
				if response.AccountsCursor == nil {
					return ListAccountsOutput{}, fmt.Errorf("payments v1 list accounts returned no cursor")
				}
				return fromV1AccountsCursor(response.AccountsCursor.Cursor), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
				response, err := sdk.Payments.V3.ListAccounts(ctx, toV3ListAccountsRequest(input))
				if err != nil {
					return ListAccountsOutput{}, err
				}
				if response.V3AccountsCursorResponse == nil {
					return ListAccountsOutput{}, fmt.Errorf("payments v3 list accounts returned no cursor")
				}
				return fromV3AccountsCursor(response.V3AccountsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetAccountHandlers(sdk *formance.Formance) []GetAccountHandler {
	return []GetAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
				response, err := sdk.Payments.V1.PaymentsgetAccount(ctx, operations.PaymentsgetAccountRequest{
					AccountID: input.AccountID,
				})
				if err != nil {
					return GetAccountOutput{}, err
				}
				if response.PaymentsAccountResponse == nil {
					return GetAccountOutput{}, fmt.Errorf("payments v1 get account returned no data")
				}
				return GetAccountOutput{Account: fromV1Account(response.PaymentsAccountResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
				response, err := sdk.Payments.V3.GetAccount(ctx, operations.V3GetAccountRequest{
					AccountID: input.AccountID,
				})
				if err != nil {
					return GetAccountOutput{}, err
				}
				if response.V3GetAccountResponse == nil {
					return GetAccountOutput{}, fmt.Errorf("payments v3 get account returned no data")
				}
				return GetAccountOutput{Account: fromV3Account(response.V3GetAccountResponse.Data)}, nil
			},
		},
	}
}

func toV1ListAccountsRequest(input ListAccountsInput) operations.PaymentslistAccountsRequest {
	request := operations.PaymentslistAccountsRequest{
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	return request
}

func toV3ListAccountsRequest(input ListAccountsInput) operations.V3ListAccountsRequest {
	request := operations.V3ListAccountsRequest{
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	return request
}

func fromV1AccountsCursor(cursor shared.Cursor) ListAccountsOutput {
	accounts := make([]AccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, fromV1Account(account))
	}
	return ListAccountsOutput{
		Accounts: accounts,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV3AccountsCursor(cursor shared.V3AccountsCursorResponseCursor) ListAccountsOutput {
	accounts := make([]AccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, fromV3Account(account))
	}
	return ListAccountsOutput{
		Accounts: accounts,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV1Account(account shared.PaymentsAccount) AccountSummary {
	provider := ""
	if account.Provider != nil {
		provider = *account.Provider
	}
	return AccountSummary{
		ID:              account.ID,
		Reference:       account.Reference,
		Name:            account.AccountName,
		CreatedAt:       account.CreatedAt,
		ConnectorID:     account.ConnectorID,
		DefaultAsset:    account.DefaultAsset,
		DefaultCurrency: account.DefaultCurrency,
		Provider:        provider,
		Type:            string(account.Type),
		Metadata:        account.Metadata,
	}
}

func fromV3Account(account shared.V3Account) AccountSummary {
	name := ""
	if account.Name != nil {
		name = *account.Name
	}
	defaultAsset := ""
	if account.DefaultAsset != nil {
		defaultAsset = *account.DefaultAsset
	}
	return AccountSummary{
		ID:           account.ID,
		Reference:    account.Reference,
		Name:         name,
		CreatedAt:    account.CreatedAt,
		ConnectorID:  account.ConnectorID,
		DefaultAsset: defaultAsset,
		Provider:     account.Provider,
		Type:         string(account.Type),
		Metadata:     account.Metadata,
	}
}

func pointer[T any](value T) *T {
	return &value
}
