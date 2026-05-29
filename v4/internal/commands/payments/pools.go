package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	FeatureAddAccountToPool      capabilities.Feature = "addAccountToPool"
	FeatureCreatePool            capabilities.Feature = "createPool"
	FeatureDeletePool            capabilities.Feature = "deletePool"
	FeatureGetPool               capabilities.Feature = "getPool"
	FeatureGetPoolBalances       capabilities.Feature = "getPoolBalances"
	FeatureGetPoolBalancesLatest capabilities.Feature = "getPoolBalancesLatest"
	FeatureListPools             capabilities.Feature = "listPools"
	FeatureRemoveAccountFromPool capabilities.Feature = "removeAccountFromPool"
	FeatureUpdatePoolQuery       capabilities.Feature = "updatePoolQuery"
)

type PoolSummary struct {
	ID        string         `json:"id" yaml:"id"`
	Name      string         `json:"name" yaml:"name"`
	Accounts  []string       `json:"accounts" yaml:"accounts"`
	CreatedAt time.Time      `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	Type      string         `json:"type,omitempty" yaml:"type,omitempty"`
	Query     map[string]any `json:"query,omitempty" yaml:"query,omitempty"`
}

type ListPoolsInput struct {
	PageSize int64
	Cursor   string
	Query    string
}

type ListPoolsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Pools      []PoolSummary           `json:"pools" yaml:"pools"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type CreatePoolInput struct {
	Payload []byte
}

type CreatePoolOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PoolID     string                  `json:"poolID" yaml:"poolID"`
}

type GetPoolInput struct {
	PoolID string
}

type GetPoolOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Pool       PoolSummary             `json:"pool" yaml:"pool"`
}

type DeletePoolInput struct {
	PoolID string
}

type DeletePoolOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PoolID     string                  `json:"poolID" yaml:"poolID"`
}

type PoolAccountInput struct {
	PoolID    string
	AccountID string
}

type PoolAccountOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PoolID     string                  `json:"poolID" yaml:"poolID"`
	AccountID  string                  `json:"accountID" yaml:"accountID"`
}

type UpdatePoolQueryInput struct {
	PoolID string
	Query  map[string]any
}

type UpdatePoolQueryOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PoolID     string                  `json:"poolID" yaml:"poolID"`
}

type PoolBalanceSummary struct {
	Asset           string   `json:"asset" yaml:"asset"`
	Amount          string   `json:"amount" yaml:"amount"`
	RelatedAccounts []string `json:"relatedAccounts,omitempty" yaml:"relatedAccounts,omitempty"`
}

type GetPoolBalancesInput struct {
	PoolID string
	At     time.Time
	Latest bool
}

type GetPoolBalancesOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PoolID     string                  `json:"poolID" yaml:"poolID"`
	Balances   []PoolBalanceSummary    `json:"balances" yaml:"balances"`
}

type ListPoolsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListPoolsInput) (ListPoolsOutput, error)
}

type CreatePoolHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreatePoolInput) (CreatePoolOutput, error)
}

type GetPoolHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetPoolInput) (GetPoolOutput, error)
}

type DeletePoolHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeletePoolInput) (DeletePoolOutput, error)
}

type PoolAccountHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, PoolAccountInput) (PoolAccountOutput, error)
}

type UpdatePoolQueryHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UpdatePoolQueryInput) (UpdatePoolQueryOutput, error)
}

type GetPoolBalancesHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetPoolBalancesInput) (GetPoolBalancesOutput, error)
}

type ListPoolsService struct {
	Handlers []ListPoolsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreatePoolService struct {
	Handlers []CreatePoolHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetPoolService struct {
	Handlers []GetPoolHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeletePoolService struct {
	Handlers []DeletePoolHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type AddAccountToPoolService struct {
	Handlers []PoolAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type RemoveAccountFromPoolService struct {
	Handlers []PoolAccountHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UpdatePoolQueryService struct {
	Handlers []UpdatePoolQueryHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetPoolBalancesService struct {
	Handlers []GetPoolBalancesHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListPoolsService) Run(ctx context.Context, input ListPoolsInput) (ListPoolsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListPoolsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListPoolsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListPoolsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListPoolsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreatePoolService) Run(ctx context.Context, input CreatePoolInput) (CreatePoolOutput, error) {
	if len(input.Payload) == 0 {
		return CreatePoolOutput{}, fmt.Errorf("pool request is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreatePoolHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreatePoolOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreatePoolOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreatePoolOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetPoolService) Run(ctx context.Context, input GetPoolInput) (GetPoolOutput, error) {
	if input.PoolID == "" {
		return GetPoolOutput{}, fmt.Errorf("pool id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetPoolHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetPoolOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetPoolOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetPoolOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s DeletePoolService) Run(ctx context.Context, input DeletePoolInput) (DeletePoolOutput, error) {
	if input.PoolID == "" {
		return DeletePoolOutput{}, fmt.Errorf("pool id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeletePoolHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeletePoolOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeletePoolOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeletePoolOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s AddAccountToPoolService) Run(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
	return runPoolAccountService(ctx, input, s.Handlers, s.Resolve)
}

func (s RemoveAccountFromPoolService) Run(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
	return runPoolAccountService(ctx, input, s.Handlers, s.Resolve)
}

func (s UpdatePoolQueryService) Run(ctx context.Context, input UpdatePoolQueryInput) (UpdatePoolQueryOutput, error) {
	if input.PoolID == "" {
		return UpdatePoolQueryOutput{}, fmt.Errorf("pool id is required")
	}
	if input.Query == nil {
		return UpdatePoolQueryOutput{}, fmt.Errorf("pool query is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]UpdatePoolQueryHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return UpdatePoolQueryOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return UpdatePoolQueryOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return UpdatePoolQueryOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetPoolBalancesService) Run(ctx context.Context, input GetPoolBalancesInput) (GetPoolBalancesOutput, error) {
	if input.PoolID == "" {
		return GetPoolBalancesOutput{}, fmt.Errorf("pool id is required")
	}
	if !input.Latest && input.At.IsZero() {
		return GetPoolBalancesOutput{}, fmt.Errorf("pool balances at time is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetPoolBalancesHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetPoolBalancesOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetPoolBalancesOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetPoolBalancesOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runPoolAccountService(
	ctx context.Context,
	input PoolAccountInput,
	serviceHandlers []PoolAccountHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (PoolAccountOutput, error) {
	if input.PoolID == "" {
		return PoolAccountOutput{}, fmt.Errorf("pool id is required")
	}
	if input.AccountID == "" {
		return PoolAccountOutput{}, fmt.Errorf("account id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(serviceHandlers))
	handlers := map[capabilities.APIVersion]PoolAccountHandler{}
	for _, handler := range serviceHandlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return PoolAccountOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return PoolAccountOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return PoolAccountOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListPoolsHandlers(sdk *formance.Formance) []ListPoolsHandler {
	return []ListPoolsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListPoolsInput) (ListPoolsOutput, error) {
				response, err := sdk.Payments.V1.ListPools(ctx, operations.ListPoolsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Query:    optionalString(input.Query),
				})
				if err != nil {
					return ListPoolsOutput{}, err
				}
				if response.PoolsCursor == nil {
					return ListPoolsOutput{}, fmt.Errorf("payments v1 list pools returned no cursor")
				}
				return fromV1PoolsCursor(response.PoolsCursor.Cursor), nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input ListPoolsInput) (ListPoolsOutput, error) {
				response, err := sdk.Payments.V3.ListPools(ctx, operations.V3ListPoolsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListPoolsOutput{}, err
				}
				if response.V3PoolsCursorResponse == nil {
					return ListPoolsOutput{}, fmt.Errorf("payments v3 list pools returned no cursor")
				}
				return fromV3PoolsCursor(response.V3PoolsCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKCreatePoolHandlers(sdk *formance.Formance) []CreatePoolHandler {
	return []CreatePoolHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreatePoolInput) (CreatePoolOutput, error) {
				var request shared.PoolRequest
				if err := json.Unmarshal(input.Payload, &request); err != nil {
					return CreatePoolOutput{}, err
				}
				response, err := sdk.Payments.V1.CreatePool(ctx, request)
				if err != nil {
					return CreatePoolOutput{}, err
				}
				if response.PoolResponse == nil {
					return CreatePoolOutput{}, fmt.Errorf("payments v1 create pool returned no data")
				}
				return CreatePoolOutput{PoolID: response.PoolResponse.Data.ID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input CreatePoolInput) (CreatePoolOutput, error) {
				var request shared.V3CreatePoolRequest
				if err := json.Unmarshal(input.Payload, &request); err != nil {
					return CreatePoolOutput{}, err
				}
				response, err := sdk.Payments.V3.CreatePool(ctx, &request)
				if err != nil {
					return CreatePoolOutput{}, err
				}
				if response.V3CreatePoolResponse == nil {
					return CreatePoolOutput{}, fmt.Errorf("payments v3 create pool returned no data")
				}
				return CreatePoolOutput{PoolID: response.V3CreatePoolResponse.Data}, nil
			},
		},
	}
}

func SDKGetPoolHandlers(sdk *formance.Formance) []GetPoolHandler {
	return []GetPoolHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetPoolInput) (GetPoolOutput, error) {
				response, err := sdk.Payments.V1.GetPool(ctx, operations.GetPoolRequest{PoolID: input.PoolID})
				if err != nil {
					return GetPoolOutput{}, err
				}
				if response.PoolResponse == nil {
					return GetPoolOutput{}, fmt.Errorf("payments v1 get pool returned no data")
				}
				return GetPoolOutput{Pool: fromV1Pool(response.PoolResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetPoolInput) (GetPoolOutput, error) {
				response, err := sdk.Payments.V3.GetPool(ctx, operations.V3GetPoolRequest{PoolID: input.PoolID})
				if err != nil {
					return GetPoolOutput{}, err
				}
				if response.V3GetPoolResponse == nil {
					return GetPoolOutput{}, fmt.Errorf("payments v3 get pool returned no data")
				}
				return GetPoolOutput{Pool: fromV3Pool(response.V3GetPoolResponse.Data)}, nil
			},
		},
	}
}

func SDKDeletePoolHandlers(sdk *formance.Formance) []DeletePoolHandler {
	return []DeletePoolHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeletePoolInput) (DeletePoolOutput, error) {
				if _, err := sdk.Payments.V1.DeletePool(ctx, operations.DeletePoolRequest{PoolID: input.PoolID}); err != nil {
					return DeletePoolOutput{}, err
				}
				return DeletePoolOutput{PoolID: input.PoolID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input DeletePoolInput) (DeletePoolOutput, error) {
				if _, err := sdk.Payments.V3.DeletePool(ctx, operations.V3DeletePoolRequest{PoolID: input.PoolID}); err != nil {
					return DeletePoolOutput{}, err
				}
				return DeletePoolOutput{PoolID: input.PoolID}, nil
			},
		},
	}
}

func SDKAddAccountToPoolHandlers(sdk *formance.Formance) []PoolAccountHandler {
	return []PoolAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
				if _, err := sdk.Payments.V1.AddAccountToPool(ctx, operations.AddAccountToPoolRequest{
					PoolID: input.PoolID,
					AddAccountToPoolRequest: shared.AddAccountToPoolRequest{
						AccountID: input.AccountID,
					},
				}); err != nil {
					return PoolAccountOutput{}, err
				}
				return PoolAccountOutput{PoolID: input.PoolID, AccountID: input.AccountID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
				if _, err := sdk.Payments.V3.AddAccountToPool(ctx, operations.V3AddAccountToPoolRequest{
					PoolID:    input.PoolID,
					AccountID: input.AccountID,
				}); err != nil {
					return PoolAccountOutput{}, err
				}
				return PoolAccountOutput{PoolID: input.PoolID, AccountID: input.AccountID}, nil
			},
		},
	}
}

func SDKRemoveAccountFromPoolHandlers(sdk *formance.Formance) []PoolAccountHandler {
	return []PoolAccountHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
				if _, err := sdk.Payments.V1.RemoveAccountFromPool(ctx, operations.RemoveAccountFromPoolRequest{
					PoolID:    input.PoolID,
					AccountID: input.AccountID,
				}); err != nil {
					return PoolAccountOutput{}, err
				}
				return PoolAccountOutput{PoolID: input.PoolID, AccountID: input.AccountID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input PoolAccountInput) (PoolAccountOutput, error) {
				if _, err := sdk.Payments.V3.RemoveAccountFromPool(ctx, operations.V3RemoveAccountFromPoolRequest{
					PoolID:    input.PoolID,
					AccountID: input.AccountID,
				}); err != nil {
					return PoolAccountOutput{}, err
				}
				return PoolAccountOutput{PoolID: input.PoolID, AccountID: input.AccountID}, nil
			},
		},
	}
}

func SDKUpdatePoolQueryHandlers(sdk *formance.Formance) []UpdatePoolQueryHandler {
	return []UpdatePoolQueryHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input UpdatePoolQueryInput) (UpdatePoolQueryOutput, error) {
				if _, err := sdk.Payments.V1.UpdatePoolQuery(ctx, operations.UpdatePoolQueryRequest{
					PoolID: input.PoolID,
					UpdatePoolQueryRequest: shared.UpdatePoolQueryRequest{
						Query: input.Query,
					},
				}); err != nil {
					return UpdatePoolQueryOutput{}, err
				}
				return UpdatePoolQueryOutput{PoolID: input.PoolID}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input UpdatePoolQueryInput) (UpdatePoolQueryOutput, error) {
				if _, err := sdk.Payments.V3.UpdatePoolQuery(ctx, operations.V3UpdatePoolQueryRequest{
					PoolID: input.PoolID,
					V3UpdatePoolQueryRequest: &shared.V3UpdatePoolQueryRequest{
						Query: input.Query,
					},
				}); err != nil {
					return UpdatePoolQueryOutput{}, err
				}
				return UpdatePoolQueryOutput{PoolID: input.PoolID}, nil
			},
		},
	}
}

func SDKGetPoolBalancesHandlers(sdk *formance.Formance) []GetPoolBalancesHandler {
	return []GetPoolBalancesHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetPoolBalancesInput) (GetPoolBalancesOutput, error) {
				if input.Latest {
					response, err := sdk.Payments.V1.GetPoolBalancesLatest(ctx, operations.GetPoolBalancesLatestRequest{PoolID: input.PoolID})
					if err != nil {
						return GetPoolBalancesOutput{}, err
					}
					if response.PoolBalancesLatestResponse == nil {
						return GetPoolBalancesOutput{}, fmt.Errorf("payments v1 get latest pool balances returned no data")
					}
					return GetPoolBalancesOutput{PoolID: input.PoolID, Balances: fromV1PoolBalances(response.PoolBalancesLatestResponse.Data)}, nil
				}
				response, err := sdk.Payments.V1.GetPoolBalances(ctx, operations.GetPoolBalancesRequest{
					PoolID: input.PoolID,
					At:     input.At,
				})
				if err != nil {
					return GetPoolBalancesOutput{}, err
				}
				if response.PoolBalancesResponse == nil {
					return GetPoolBalancesOutput{}, fmt.Errorf("payments v1 get pool balances returned no data")
				}
				return GetPoolBalancesOutput{PoolID: input.PoolID, Balances: fromV1PoolBalances(response.PoolBalancesResponse.Data.Balances)}, nil
			},
		},
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetPoolBalancesInput) (GetPoolBalancesOutput, error) {
				if input.Latest {
					response, err := sdk.Payments.V3.GetPoolBalancesLatest(ctx, operations.V3GetPoolBalancesLatestRequest{PoolID: input.PoolID})
					if err != nil {
						return GetPoolBalancesOutput{}, err
					}
					if response.V3PoolBalancesResponse == nil {
						return GetPoolBalancesOutput{}, fmt.Errorf("payments v3 get latest pool balances returned no data")
					}
					return GetPoolBalancesOutput{PoolID: input.PoolID, Balances: fromV3PoolBalances(response.V3PoolBalancesResponse.Data)}, nil
				}
				response, err := sdk.Payments.V3.GetPoolBalances(ctx, operations.V3GetPoolBalancesRequest{
					PoolID: input.PoolID,
					At:     &input.At,
				})
				if err != nil {
					return GetPoolBalancesOutput{}, err
				}
				if response.V3PoolBalancesResponse == nil {
					return GetPoolBalancesOutput{}, fmt.Errorf("payments v3 get pool balances returned no data")
				}
				return GetPoolBalancesOutput{PoolID: input.PoolID, Balances: fromV3PoolBalances(response.V3PoolBalancesResponse.Data)}, nil
			},
		},
	}
}

func fromV1PoolsCursor(cursor shared.PoolsCursorCursor) ListPoolsOutput {
	pools := make([]PoolSummary, 0, len(cursor.Data))
	for _, pool := range cursor.Data {
		pools = append(pools, fromV1Pool(pool))
	}
	return ListPoolsOutput{Pools: pools, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV3PoolsCursor(cursor shared.V3PoolsCursorResponseCursor) ListPoolsOutput {
	pools := make([]PoolSummary, 0, len(cursor.Data))
	for _, pool := range cursor.Data {
		pools = append(pools, fromV3Pool(pool))
	}
	return ListPoolsOutput{Pools: pools, HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous}
}

func fromV1Pool(pool shared.Pool) PoolSummary {
	summary := PoolSummary{
		ID:       pool.ID,
		Name:     pool.Name,
		Accounts: pool.Accounts,
		Query:    pool.Query,
	}
	if pool.Type != nil {
		summary.Type = string(*pool.Type)
	}
	return summary
}

func fromV3Pool(pool shared.V3Pool) PoolSummary {
	summary := PoolSummary{
		ID:        pool.ID,
		Name:      pool.Name,
		Accounts:  pool.PoolAccounts,
		CreatedAt: pool.CreatedAt,
		Query:     pool.Query,
	}
	if pool.Type != nil {
		summary.Type = string(*pool.Type)
	}
	return summary
}

func fromV1PoolBalances(balances []shared.PoolBalance) []PoolBalanceSummary {
	summaries := make([]PoolBalanceSummary, 0, len(balances))
	for _, balance := range balances {
		summaries = append(summaries, PoolBalanceSummary{
			Asset:           balance.Asset,
			Amount:          bigIntString(balance.Amount),
			RelatedAccounts: balance.RelatedAccounts,
		})
	}
	return summaries
}

func fromV3PoolBalances(balances []shared.V3PoolBalance) []PoolBalanceSummary {
	summaries := make([]PoolBalanceSummary, 0, len(balances))
	for _, balance := range balances {
		summaries = append(summaries, PoolBalanceSummary{
			Asset:           balance.Asset,
			Amount:          bigIntString(balance.Amount),
			RelatedAccounts: balance.RelatedAccounts,
		})
	}
	return summaries
}
