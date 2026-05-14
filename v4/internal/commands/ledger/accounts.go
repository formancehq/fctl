package ledger

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	FeatureListAccounts capabilities.Feature = "listAccounts"
	FeatureGetAccount   capabilities.Feature = "getAccount"
)

type ListAccountsInput struct {
	Ledger   string
	PageSize int64
	Cursor   string
	Account  string
	Metadata map[string]string
}

type ListAccountsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Accounts   []AccountSummary        `json:"accounts" yaml:"accounts"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type AccountSummary struct {
	Address  string         `json:"address" yaml:"address"`
	Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type GetAccountInput struct {
	Ledger  string
	Account string
}

type GetAccountOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Account    AccountDetail           `json:"account" yaml:"account"`
}

type AccountDetail struct {
	Address  string              `json:"address" yaml:"address"`
	Metadata map[string]any      `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Volumes  map[string]VolumeIO `json:"volumes,omitempty" yaml:"volumes,omitempty"`
}

type VolumeIO struct {
	Input   string `json:"input" yaml:"input"`
	Output  string `json:"output" yaml:"output"`
	Balance string `json:"balance,omitempty" yaml:"balance,omitempty"`
}

type ListAccountsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListAccountsInput) (ListAccountsOutput, error)
}

type ListAccountsService struct {
	Handlers []ListAccountsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListAccountsService) Run(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
	if input.Ledger == "" {
		return ListAccountsOutput{}, fmt.Errorf("ledger is required")
	}

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

type GetAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetAccountInput) (GetAccountOutput, error)
}

type GetAccountService struct {
	Handlers []GetAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s GetAccountService) Run(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
	if input.Ledger == "" {
		return GetAccountOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Account == "" {
		return GetAccountOutput{}, fmt.Errorf("account is required")
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
				response, err := sdk.Ledger.V1.ListAccounts(ctx, toV1ListAccountsRequest(input))
				if err != nil {
					return ListAccountsOutput{}, err
				}
				if response.AccountsCursorResponse == nil {
					return ListAccountsOutput{}, fmt.Errorf("ledger v1 list accounts returned no cursor")
				}
				return fromV1ListAccounts(response.AccountsCursorResponse.Cursor), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
				response, err := sdk.Ledger.V2.ListAccounts(ctx, toV2ListAccountsRequest(input))
				if err != nil {
					return ListAccountsOutput{}, err
				}
				if response.V2AccountsCursorResponse == nil {
					return ListAccountsOutput{}, fmt.Errorf("ledger v2 list accounts returned no cursor")
				}
				return fromV2ListAccounts(response.V2AccountsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetAccountHandlers(sdk *formance.Formance) []GetAccountHandler {
	return []GetAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
				response, err := sdk.Ledger.V1.GetAccount(ctx, operations.GetAccountRequest{
					Ledger:  input.Ledger,
					Address: input.Account,
				})
				if err != nil {
					return GetAccountOutput{}, err
				}
				if response.AccountResponse == nil {
					return GetAccountOutput{}, fmt.Errorf("ledger v1 get account returned no data")
				}
				return GetAccountOutput{Account: fromV1AccountDetail(response.AccountResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetAccountInput) (GetAccountOutput, error) {
				response, err := sdk.Ledger.V2.GetAccount(ctx, operations.V2GetAccountRequest{
					Ledger:  input.Ledger,
					Address: input.Account,
				})
				if err != nil {
					return GetAccountOutput{}, err
				}
				if response.V2AccountResponse == nil {
					return GetAccountOutput{}, fmt.Errorf("ledger v2 get account returned no data")
				}
				return GetAccountOutput{Account: fromV2AccountDetail(response.V2AccountResponse.Data)}, nil
			},
		},
	}
}

func toV1ListAccountsRequest(input ListAccountsInput) operations.ListAccountsRequest {
	request := operations.ListAccountsRequest{
		Ledger:   input.Ledger,
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	if input.Account != "" {
		request.Address = pointer(input.Account)
	}
	if len(input.Metadata) > 0 {
		request.Metadata = stringMapToAny(input.Metadata)
	}
	return request
}

func toV2ListAccountsRequest(input ListAccountsInput) operations.V2ListAccountsRequest {
	request := operations.V2ListAccountsRequest{
		Ledger:   input.Ledger,
		PageSize: pointer(input.PageSize),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	queryParts := make([]map[string]any, 0, len(input.Metadata)+1)
	if input.Account != "" {
		queryParts = append(queryParts, map[string]any{
			"$match": map[string]any{"address": input.Account},
		})
	}
	for key, value := range input.Metadata {
		queryParts = append(queryParts, map[string]any{
			"$match": map[string]any{"metadata[" + key + "]": value},
		})
	}
	if len(queryParts) == 1 {
		request.Query = queryParts[0]
	}
	if len(queryParts) > 1 {
		request.Query = map[string]any{"$and": queryParts}
	}
	return request
}

func fromV1ListAccounts(cursor shared.AccountsCursorResponseCursor) ListAccountsOutput {
	accounts := make([]AccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, AccountSummary{
			Address:  account.Address,
			Metadata: account.Metadata,
		})
	}
	return ListAccountsOutput{
		Accounts: accounts,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV2ListAccounts(cursor shared.V2AccountsCursorResponseCursor) ListAccountsOutput {
	accounts := make([]AccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, AccountSummary{
			Address:  account.Address,
			Metadata: stringMapToAny(account.Metadata),
		})
	}
	return ListAccountsOutput{
		Accounts: accounts,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV1AccountDetail(account shared.AccountWithVolumesAndBalances) AccountDetail {
	return AccountDetail{
		Address:  account.Address,
		Metadata: account.Metadata,
		Volumes:  fromV1Volumes(account.Volumes),
	}
}

func fromV2AccountDetail(account shared.V2Account) AccountDetail {
	return AccountDetail{
		Address:  account.Address,
		Metadata: stringMapToAny(account.Metadata),
		Volumes:  fromV2Volumes(account.Volumes),
	}
}

func fromV1Volumes(volumes map[string]shared.Volume) map[string]VolumeIO {
	if len(volumes) == 0 {
		return nil
	}
	ret := make(map[string]VolumeIO, len(volumes))
	for asset, volume := range volumes {
		ret[asset] = VolumeIO{
			Input:   bigIntString(volume.Input),
			Output:  bigIntString(volume.Output),
			Balance: bigIntString(volume.Balance),
		}
	}
	return ret
}

func fromV2Volumes(volumes map[string]shared.V2Volume) map[string]VolumeIO {
	if len(volumes) == 0 {
		return nil
	}
	ret := make(map[string]VolumeIO, len(volumes))
	for asset, volume := range volumes {
		ret[asset] = VolumeIO{
			Input:   bigIntString(volume.Input),
			Output:  bigIntString(volume.Output),
			Balance: bigIntString(volume.Balance),
		}
	}
	return ret
}
