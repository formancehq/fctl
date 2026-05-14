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
	FeatureAddAccountToPool      capabilities.Feature = "addAccountToPool"
	FeatureDeletePool            capabilities.Feature = "deletePool"
	FeatureGetPool               capabilities.Feature = "getPool"
	FeatureListPools             capabilities.Feature = "listPools"
	FeatureRemoveAccountFromPool capabilities.Feature = "removeAccountFromPool"
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

type ListPoolsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListPoolsInput) (ListPoolsOutput, error)
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

type ListPoolsService struct {
	Handlers []ListPoolsHandler
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
