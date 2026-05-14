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
	FeatureCreateBankAccount      capabilities.Feature = "createBankAccount"
	FeatureForwardBankAccount     capabilities.Feature = "forwardBankAccount"
	FeatureGetBankAccount         capabilities.Feature = "getBankAccount"
	FeatureListBankAccounts       capabilities.Feature = "listBankAccounts"
	FeatureSetBankAccountMetadata capabilities.Feature = "updateBankAccountMetadata"
)

type CreateBankAccountInput struct {
	AccountNumber string
	ConnectorID   string
	Country       string
	Iban          string
	Metadata      map[string]string
	Name          string
	SwiftBicCode  string
}

type CreateBankAccountOutput struct {
	APIVersion    capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	BankAccountID string                  `json:"bankAccountID" yaml:"bankAccountID"`
}

type ForwardBankAccountInput struct {
	BankAccountID string
	ConnectorID   string
}

type ForwardBankAccountOutput struct {
	APIVersion    capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	BankAccountID string                  `json:"bankAccountID" yaml:"bankAccountID"`
	ConnectorID   string                  `json:"connectorID" yaml:"connectorID"`
	TaskID        string                  `json:"taskID,omitempty" yaml:"taskID,omitempty"`
}

type SetBankAccountMetadataInput struct {
	BankAccountID string
	Metadata      map[string]string
}

type SetBankAccountMetadataOutput struct {
	APIVersion    capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	BankAccountID string                  `json:"bankAccountID" yaml:"bankAccountID"`
}

type BankAccountSummary struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	CreatedAt     time.Time         `json:"createdAt" yaml:"createdAt"`
	Country       string            `json:"country,omitempty" yaml:"country,omitempty"`
	AccountNumber string            `json:"accountNumber,omitempty" yaml:"accountNumber,omitempty"`
	Iban          string            `json:"iban,omitempty" yaml:"iban,omitempty"`
	SwiftBicCode  string            `json:"swiftBicCode,omitempty" yaml:"swiftBicCode,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListBankAccountsInput struct {
	PageSize int64
	Cursor   string
}

type ListBankAccountsOutput struct {
	APIVersion   capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	BankAccounts []BankAccountSummary    `json:"bankAccounts" yaml:"bankAccounts"`
	HasMore      bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize     int64                   `json:"pageSize" yaml:"pageSize"`
	Next         *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous     *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetBankAccountInput struct {
	BankAccountID string
}

type GetBankAccountOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	BankAccount BankAccountSummary      `json:"bankAccount" yaml:"bankAccount"`
}

type ListBankAccountsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListBankAccountsInput) (ListBankAccountsOutput, error)
}

type CreateBankAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateBankAccountInput) (CreateBankAccountOutput, error)
}

type ForwardBankAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ForwardBankAccountInput) (ForwardBankAccountOutput, error)
}

type SetBankAccountMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, SetBankAccountMetadataInput) (SetBankAccountMetadataOutput, error)
}

type GetBankAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetBankAccountInput) (GetBankAccountOutput, error)
}

type ListBankAccountsService struct {
	Handlers []ListBankAccountsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateBankAccountService struct {
	Handlers []CreateBankAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ForwardBankAccountService struct {
	Handlers []ForwardBankAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type SetBankAccountMetadataService struct {
	Handlers []SetBankAccountMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetBankAccountService struct {
	Handlers []GetBankAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListBankAccountsService) Run(ctx context.Context, input ListBankAccountsInput) (ListBankAccountsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListBankAccountsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListBankAccountsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListBankAccountsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListBankAccountsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreateBankAccountService) Run(ctx context.Context, input CreateBankAccountInput) (CreateBankAccountOutput, error) {
	if input.Name == "" {
		return CreateBankAccountOutput{}, fmt.Errorf("bank account name is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateBankAccountHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateBankAccountOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateBankAccountOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateBankAccountOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ForwardBankAccountService) Run(ctx context.Context, input ForwardBankAccountInput) (ForwardBankAccountOutput, error) {
	if input.BankAccountID == "" {
		return ForwardBankAccountOutput{}, fmt.Errorf("bank account id is required")
	}
	if input.ConnectorID == "" {
		return ForwardBankAccountOutput{}, fmt.Errorf("connector id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ForwardBankAccountHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ForwardBankAccountOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ForwardBankAccountOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ForwardBankAccountOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s SetBankAccountMetadataService) Run(ctx context.Context, input SetBankAccountMetadataInput) (SetBankAccountMetadataOutput, error) {
	if input.BankAccountID == "" {
		return SetBankAccountMetadataOutput{}, fmt.Errorf("bank account id is required")
	}
	if len(input.Metadata) == 0 {
		return SetBankAccountMetadataOutput{}, fmt.Errorf("bank account metadata is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]SetBankAccountMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return SetBankAccountMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return SetBankAccountMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return SetBankAccountMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetBankAccountService) Run(ctx context.Context, input GetBankAccountInput) (GetBankAccountOutput, error) {
	if input.BankAccountID == "" {
		return GetBankAccountOutput{}, fmt.Errorf("bank account id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetBankAccountHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetBankAccountOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetBankAccountOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetBankAccountOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKCreateBankAccountHandlers(sdk *formance.Formance) []CreateBankAccountHandler {
	return []CreateBankAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateBankAccountInput) (CreateBankAccountOutput, error) {
				response, err := sdk.Payments.V1.CreateBankAccount(ctx, shared.BankAccountRequest{
					AccountNumber: optionalString(input.AccountNumber),
					ConnectorID:   optionalString(input.ConnectorID),
					Country:       input.Country,
					Iban:          optionalString(input.Iban),
					Metadata:      input.Metadata,
					Name:          input.Name,
					SwiftBicCode:  optionalString(input.SwiftBicCode),
				})
				if err != nil {
					return CreateBankAccountOutput{}, err
				}
				if response.BankAccountResponse == nil {
					return CreateBankAccountOutput{}, fmt.Errorf("payments v1 create bank account returned no data")
				}
				return CreateBankAccountOutput{BankAccountID: response.BankAccountResponse.Data.ID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input CreateBankAccountInput) (CreateBankAccountOutput, error) {
				response, err := sdk.Payments.V3.CreateBankAccount(ctx, &shared.V3CreateBankAccountRequest{
					AccountNumber: optionalString(input.AccountNumber),
					Country:       optionalString(input.Country),
					Iban:          optionalString(input.Iban),
					Metadata:      input.Metadata,
					Name:          input.Name,
					SwiftBicCode:  optionalString(input.SwiftBicCode),
				})
				if err != nil {
					return CreateBankAccountOutput{}, err
				}
				if response.V3CreateBankAccountResponse == nil {
					return CreateBankAccountOutput{}, fmt.Errorf("payments v3 create bank account returned no data")
				}
				return CreateBankAccountOutput{BankAccountID: response.V3CreateBankAccountResponse.Data}, nil
			},
		},
	}
}

func SDKForwardBankAccountHandlers(sdk *formance.Formance) []ForwardBankAccountHandler {
	return []ForwardBankAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ForwardBankAccountInput) (ForwardBankAccountOutput, error) {
				response, err := sdk.Payments.V1.ForwardBankAccount(ctx, operations.ForwardBankAccountRequest{
					BankAccountID: input.BankAccountID,
					ForwardBankAccountRequest: shared.ForwardBankAccountRequest{
						ConnectorID: input.ConnectorID,
					},
				})
				if err != nil {
					return ForwardBankAccountOutput{}, err
				}
				if response.BankAccountResponse == nil {
					return ForwardBankAccountOutput{}, fmt.Errorf("payments v1 forward bank account returned no data")
				}
				bankAccount := response.BankAccountResponse.Data
				return ForwardBankAccountOutput{
					BankAccountID: bankAccount.ID,
					ConnectorID:   stringValue(bankAccount.ConnectorID),
				}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ForwardBankAccountInput) (ForwardBankAccountOutput, error) {
				response, err := sdk.Payments.V3.ForwardBankAccount(ctx, operations.V3ForwardBankAccountRequest{
					BankAccountID: input.BankAccountID,
					V3ForwardBankAccountRequest: &shared.V3ForwardBankAccountRequest{
						ConnectorID: input.ConnectorID,
					},
				})
				if err != nil {
					return ForwardBankAccountOutput{}, err
				}
				if response.V3ForwardBankAccountResponse == nil {
					return ForwardBankAccountOutput{}, fmt.Errorf("payments v3 forward bank account returned no data")
				}
				return ForwardBankAccountOutput{
					BankAccountID: input.BankAccountID,
					ConnectorID:   input.ConnectorID,
					TaskID:        response.V3ForwardBankAccountResponse.Data.TaskID,
				}, nil
			},
		},
	}
}

func SDKSetBankAccountMetadataHandlers(sdk *formance.Formance) []SetBankAccountMetadataHandler {
	return []SetBankAccountMetadataHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input SetBankAccountMetadataInput) (SetBankAccountMetadataOutput, error) {
				if _, err := sdk.Payments.V1.UpdateBankAccountMetadata(ctx, operations.UpdateBankAccountMetadataRequest{
					BankAccountID: input.BankAccountID,
					UpdateBankAccountMetadataRequest: shared.UpdateBankAccountMetadataRequest{
						Metadata: input.Metadata,
					},
				}); err != nil {
					return SetBankAccountMetadataOutput{}, err
				}
				return SetBankAccountMetadataOutput{BankAccountID: input.BankAccountID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input SetBankAccountMetadataInput) (SetBankAccountMetadataOutput, error) {
				if _, err := sdk.Payments.V3.UpdateBankAccountMetadata(ctx, operations.V3UpdateBankAccountMetadataRequest{
					BankAccountID: input.BankAccountID,
					V3UpdateBankAccountMetadataRequest: &shared.V3UpdateBankAccountMetadataRequest{
						Metadata: input.Metadata,
					},
				}); err != nil {
					return SetBankAccountMetadataOutput{}, err
				}
				return SetBankAccountMetadataOutput{BankAccountID: input.BankAccountID}, nil
			},
		},
	}
}

func SDKListBankAccountsHandlers(sdk *formance.Formance) []ListBankAccountsHandler {
	return []ListBankAccountsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListBankAccountsInput) (ListBankAccountsOutput, error) {
				response, err := sdk.Payments.V1.ListBankAccounts(ctx, operations.ListBankAccountsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListBankAccountsOutput{}, err
				}
				if response.BankAccountsCursor == nil {
					return ListBankAccountsOutput{}, fmt.Errorf("payments v1 list bank accounts returned no cursor")
				}
				return fromV1BankAccountsCursor(response.BankAccountsCursor.Cursor), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListBankAccountsInput) (ListBankAccountsOutput, error) {
				response, err := sdk.Payments.V3.ListBankAccounts(ctx, operations.V3ListBankAccountsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListBankAccountsOutput{}, err
				}
				if response.V3BankAccountsCursorResponse == nil {
					return ListBankAccountsOutput{}, fmt.Errorf("payments v3 list bank accounts returned no cursor")
				}
				return fromV3BankAccountsCursor(response.V3BankAccountsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetBankAccountHandlers(sdk *formance.Formance) []GetBankAccountHandler {
	return []GetBankAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetBankAccountInput) (GetBankAccountOutput, error) {
				response, err := sdk.Payments.V1.GetBankAccount(ctx, operations.GetBankAccountRequest{
					BankAccountID: input.BankAccountID,
				})
				if err != nil {
					return GetBankAccountOutput{}, err
				}
				if response.BankAccountResponse == nil {
					return GetBankAccountOutput{}, fmt.Errorf("payments v1 get bank account returned no data")
				}
				return GetBankAccountOutput{BankAccount: fromV1BankAccount(response.BankAccountResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetBankAccountInput) (GetBankAccountOutput, error) {
				response, err := sdk.Payments.V3.GetBankAccount(ctx, operations.V3GetBankAccountRequest{
					BankAccountID: input.BankAccountID,
				})
				if err != nil {
					return GetBankAccountOutput{}, err
				}
				if response.V3GetBankAccountResponse == nil {
					return GetBankAccountOutput{}, fmt.Errorf("payments v3 get bank account returned no data")
				}
				return GetBankAccountOutput{BankAccount: fromV3BankAccount(response.V3GetBankAccountResponse.Data)}, nil
			},
		},
	}
}

func fromV1BankAccountsCursor(cursor shared.BankAccountsCursorCursor) ListBankAccountsOutput {
	accounts := make([]BankAccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, fromV1BankAccount(account))
	}
	return ListBankAccountsOutput{BankAccounts: accounts, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV3BankAccountsCursor(cursor shared.V3BankAccountsCursorResponseCursor) ListBankAccountsOutput {
	accounts := make([]BankAccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, fromV3BankAccount(account))
	}
	return ListBankAccountsOutput{BankAccounts: accounts, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV1BankAccount(account shared.BankAccount) BankAccountSummary {
	return BankAccountSummary{
		ID:            account.ID,
		Name:          account.Name,
		CreatedAt:     account.CreatedAt,
		Country:       account.Country,
		AccountNumber: stringValue(account.AccountNumber),
		Iban:          stringValue(account.Iban),
		SwiftBicCode:  stringValue(account.SwiftBicCode),
		Metadata:      account.Metadata,
	}
}

func fromV3BankAccount(account shared.V3BankAccount) BankAccountSummary {
	return BankAccountSummary{
		ID:            account.ID,
		Name:          account.Name,
		CreatedAt:     account.CreatedAt,
		Country:       stringValue(account.Country),
		AccountNumber: stringValue(account.AccountNumber),
		Iban:          stringValue(account.Iban),
		SwiftBicCode:  stringValue(account.SwiftBicCode),
		Metadata:      account.Metadata,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
