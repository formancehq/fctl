package wallets

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
	ProductWallets      capabilities.Product = "wallets"
	FeatureCreateWallet capabilities.Feature = "createWallet"
	FeatureGetWallet    capabilities.Feature = "getWallet"
	FeatureListWallets  capabilities.Feature = "listWallets"
	FeatureUpdateWallet capabilities.Feature = "updateWallet"
)

type WalletSummary struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Ledger    string            `json:"ledger" yaml:"ledger"`
	CreatedAt time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type CreateWalletInput struct {
	Name           string
	Metadata       map[string]string
	IdempotencyKey string
}

type CreateWalletOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID   string                  `json:"walletID" yaml:"walletID"`
}

type ListWalletsInput struct {
	PageSize int64
	Cursor   string
	Name     string
	Metadata map[string]string
}

type ListWalletsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Wallets    []WalletSummary         `json:"wallets" yaml:"wallets"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetWalletInput struct {
	WalletID string
}

type GetWalletOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Wallet     WalletSummary           `json:"wallet" yaml:"wallet"`
}

type UpdateWalletInput struct {
	WalletID       string
	Metadata       map[string]string
	IdempotencyKey string
}

type UpdateWalletOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID   string                  `json:"walletID" yaml:"walletID"`
}

type CreateWalletHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateWalletInput) (CreateWalletOutput, error)
}

type ListWalletsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListWalletsInput) (ListWalletsOutput, error)
}

type GetWalletHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetWalletInput) (GetWalletOutput, error)
}

type UpdateWalletHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UpdateWalletInput) (UpdateWalletOutput, error)
}

type CreateWalletService struct {
	Handlers []CreateWalletHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListWalletsService struct {
	Handlers []ListWalletsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetWalletService struct {
	Handlers []GetWalletHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UpdateWalletService struct {
	Handlers []UpdateWalletHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s CreateWalletService) Run(ctx context.Context, input CreateWalletInput) (CreateWalletOutput, error) {
	if input.Name == "" {
		return CreateWalletOutput{}, fmt.Errorf("wallet name is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateWalletHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateWalletOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateWalletOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateWalletOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ListWalletsService) Run(ctx context.Context, input ListWalletsInput) (ListWalletsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListWalletsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListWalletsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListWalletsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListWalletsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetWalletService) Run(ctx context.Context, input GetWalletInput) (GetWalletOutput, error) {
	if input.WalletID == "" {
		return GetWalletOutput{}, fmt.Errorf("wallet id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetWalletHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetWalletOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetWalletOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetWalletOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s UpdateWalletService) Run(ctx context.Context, input UpdateWalletInput) (UpdateWalletOutput, error) {
	if input.WalletID == "" {
		return UpdateWalletOutput{}, fmt.Errorf("wallet id is required")
	}
	if input.Metadata == nil {
		return UpdateWalletOutput{}, fmt.Errorf("wallet metadata is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]UpdateWalletHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return UpdateWalletOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return UpdateWalletOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return UpdateWalletOutput{}, err
	}
	output.APIVersion = selected
	if output.WalletID == "" {
		output.WalletID = input.WalletID
	}
	return output, nil
}

func SDKCreateWalletHandlers(sdk *formance.Formance) []CreateWalletHandler {
	return []CreateWalletHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateWalletInput) (CreateWalletOutput, error) {
				response, err := sdk.Wallets.V1.CreateWallet(ctx, operations.CreateWalletRequest{
					CreateWalletRequest: &shared.CreateWalletRequest{
						Name:     input.Name,
						Metadata: input.Metadata,
					},
					IdempotencyKey: optionalString(input.IdempotencyKey),
				})
				if err != nil {
					return CreateWalletOutput{}, err
				}
				if response.CreateWalletResponse == nil {
					return CreateWalletOutput{}, fmt.Errorf("wallets v1 create wallet returned no data")
				}
				return CreateWalletOutput{WalletID: response.CreateWalletResponse.Data.ID}, nil
			},
		},
	}
}

func SDKListWalletsHandlers(sdk *formance.Formance) []ListWalletsHandler {
	return []ListWalletsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListWalletsInput) (ListWalletsOutput, error) {
				response, err := sdk.Wallets.V1.ListWallets(ctx, operations.ListWalletsRequest{
					PageSize: optionalInt64(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Name:     optionalString(input.Name),
					Metadata: input.Metadata,
				})
				if err != nil {
					return ListWalletsOutput{}, err
				}
				if response.ListWalletsResponse == nil {
					return ListWalletsOutput{}, fmt.Errorf("wallets v1 list wallets returned no cursor")
				}
				return fromWalletsCursor(response.ListWalletsResponse.Cursor), nil
			},
		},
	}
}

func SDKGetWalletHandlers(sdk *formance.Formance) []GetWalletHandler {
	return []GetWalletHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetWalletInput) (GetWalletOutput, error) {
				response, err := sdk.Wallets.V1.GetWallet(ctx, operations.GetWalletRequest{ID: input.WalletID})
				if err != nil {
					return GetWalletOutput{}, err
				}
				if response.GetWalletResponse == nil {
					return GetWalletOutput{}, fmt.Errorf("wallets v1 get wallet returned no data")
				}
				return GetWalletOutput{Wallet: fromWalletWithBalances(response.GetWalletResponse.Data)}, nil
			},
		},
	}
}

func SDKUpdateWalletHandlers(sdk *formance.Formance) []UpdateWalletHandler {
	return []UpdateWalletHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input UpdateWalletInput) (UpdateWalletOutput, error) {
				_, err := sdk.Wallets.V1.UpdateWallet(ctx, operations.UpdateWalletRequest{
					ID:             input.WalletID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
					RequestBody: &operations.UpdateWalletRequestBody{
						Metadata: input.Metadata,
					},
				})
				if err != nil {
					return UpdateWalletOutput{}, err
				}
				return UpdateWalletOutput{WalletID: input.WalletID}, nil
			},
		},
	}
}

func fromWalletsCursor(cursor shared.ListWalletsResponseCursor) ListWalletsOutput {
	wallets := make([]WalletSummary, 0, len(cursor.Data))
	for _, wallet := range cursor.Data {
		wallets = append(wallets, fromWallet(wallet))
	}
	hasMore := false
	if cursor.HasMore != nil {
		hasMore = *cursor.HasMore
	}
	return ListWalletsOutput{
		Wallets:  wallets,
		HasMore:  hasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromWallet(wallet shared.Wallet) WalletSummary {
	return WalletSummary{
		ID:        wallet.ID,
		Name:      wallet.Name,
		Ledger:    wallet.Ledger,
		CreatedAt: wallet.CreatedAt,
		Metadata:  wallet.Metadata,
	}
}

func fromWalletWithBalances(wallet shared.WalletWithBalances) WalletSummary {
	return WalletSummary{
		ID:        wallet.ID,
		Name:      wallet.Name,
		Ledger:    wallet.Ledger,
		CreatedAt: wallet.CreatedAt,
		Metadata:  wallet.Metadata,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
