package wallets

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
	ProductWallets       capabilities.Product = "wallets"
	FeatureCreateWallet  capabilities.Feature = "createWallet"
	FeatureCreateBalance capabilities.Feature = "createBalance"
	FeatureCreditWallet  capabilities.Feature = "creditWallet"
	FeatureDebitWallet   capabilities.Feature = "debitWallet"
	FeatureConfirmHold   capabilities.Feature = "confirmHold"
	FeatureGetBalance    capabilities.Feature = "getBalance"
	FeatureGetHold       capabilities.Feature = "getHold"
	FeatureGetWallet     capabilities.Feature = "getWallet"
	FeatureListBalances  capabilities.Feature = "listBalances"
	FeatureListHolds     capabilities.Feature = "listHolds"
	FeatureListWallets   capabilities.Feature = "listWallets"
	FeatureUpdateWallet  capabilities.Feature = "updateWallet"
	FeatureVoidHold      capabilities.Feature = "voidHold"
)

type WalletSummary struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Ledger    string            `json:"ledger" yaml:"ledger"`
	CreatedAt time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type BalanceSummary struct {
	Name      string              `json:"name" yaml:"name"`
	Priority  string              `json:"priority,omitempty" yaml:"priority,omitempty"`
	ExpiresAt *time.Time          `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
	Assets    map[string]*big.Int `json:"assets,omitempty" yaml:"assets,omitempty"`
}

type HoldSummary struct {
	ID             string            `json:"id" yaml:"id"`
	WalletID       string            `json:"walletID" yaml:"walletID"`
	Asset          string            `json:"asset" yaml:"asset"`
	Description    string            `json:"description,omitempty" yaml:"description,omitempty"`
	OriginalAmount string            `json:"originalAmount,omitempty" yaml:"originalAmount,omitempty"`
	Remaining      string            `json:"remaining,omitempty" yaml:"remaining,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
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

type WalletMovementInput struct {
	WalletID       string
	Amount         *big.Int
	Asset          string
	Balance        string
	Metadata       map[string]string
	IdempotencyKey string
	Description    string
	Pending        bool
}

type WalletMovementOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID   string                  `json:"walletID" yaml:"walletID"`
	HoldID     string                  `json:"holdID,omitempty" yaml:"holdID,omitempty"`
}

type CreateBalanceInput struct {
	WalletID       string
	Name           string
	Priority       *big.Int
	ExpiresAt      *time.Time
	IdempotencyKey string
}

type CreateBalanceOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID    string                  `json:"walletID" yaml:"walletID"`
	BalanceName string                  `json:"balanceName" yaml:"balanceName"`
}

type ListBalancesInput struct {
	WalletID string
}

type ListBalancesOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID   string                  `json:"walletID" yaml:"walletID"`
	Balances   []BalanceSummary        `json:"balances" yaml:"balances"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetBalanceInput struct {
	WalletID    string
	BalanceName string
}

type GetBalanceOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WalletID   string                  `json:"walletID" yaml:"walletID"`
	Balance    BalanceSummary          `json:"balance" yaml:"balance"`
}

type ListHoldsInput struct {
	PageSize int64
	Cursor   string
	WalletID string
	Metadata map[string]string
}

type ListHoldsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Holds      []HoldSummary           `json:"holds" yaml:"holds"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetHoldInput struct {
	HoldID string
}

type GetHoldOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Hold       HoldSummary             `json:"hold" yaml:"hold"`
}

type HoldActionInput struct {
	HoldID         string
	Amount         *big.Int
	Final          *bool
	IdempotencyKey string
}

type HoldActionOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	HoldID     string                  `json:"holdID" yaml:"holdID"`
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

type WalletMovementHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, WalletMovementInput) (WalletMovementOutput, error)
}

type CreateBalanceHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateBalanceInput) (CreateBalanceOutput, error)
}

type ListBalancesHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListBalancesInput) (ListBalancesOutput, error)
}

type GetBalanceHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetBalanceInput) (GetBalanceOutput, error)
}

type ListHoldsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListHoldsInput) (ListHoldsOutput, error)
}

type GetHoldHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetHoldInput) (GetHoldOutput, error)
}

type HoldActionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, HoldActionInput) (HoldActionOutput, error)
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

type CreditWalletService struct {
	Handlers []WalletMovementHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DebitWalletService struct {
	Handlers []WalletMovementHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateBalanceService struct {
	Handlers []CreateBalanceHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListBalancesService struct {
	Handlers []ListBalancesHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetBalanceService struct {
	Handlers []GetBalanceHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListHoldsService struct {
	Handlers []ListHoldsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetHoldService struct {
	Handlers []GetHoldHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type VoidHoldService struct {
	Handlers []HoldActionHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ConfirmHoldService struct {
	Handlers []HoldActionHandler
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

func (s CreditWalletService) Run(ctx context.Context, input WalletMovementInput) (WalletMovementOutput, error) {
	return runWalletMovementService(ctx, input, s.Handlers, s.Resolve)
}

func (s DebitWalletService) Run(ctx context.Context, input WalletMovementInput) (WalletMovementOutput, error) {
	return runWalletMovementService(ctx, input, s.Handlers, s.Resolve)
}

func runWalletMovementService(
	ctx context.Context,
	input WalletMovementInput,
	handlers []WalletMovementHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (WalletMovementOutput, error) {
	if input.WalletID == "" {
		return WalletMovementOutput{}, fmt.Errorf("wallet id is required")
	}
	if input.Amount == nil {
		return WalletMovementOutput{}, fmt.Errorf("amount is required")
	}
	if input.Asset == "" {
		return WalletMovementOutput{}, fmt.Errorf("asset is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(handlers))
	handlersByVersion := map[capabilities.APIVersion]WalletMovementHandler{}
	for _, handler := range handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlersByVersion[handler.APIVersion] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return WalletMovementOutput{}, err
	}
	handler, ok := handlersByVersion[selected]
	if !ok {
		return WalletMovementOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return WalletMovementOutput{}, err
	}
	output.APIVersion = selected
	if output.WalletID == "" {
		output.WalletID = input.WalletID
	}
	return output, nil
}

func (s CreateBalanceService) Run(ctx context.Context, input CreateBalanceInput) (CreateBalanceOutput, error) {
	if input.WalletID == "" {
		return CreateBalanceOutput{}, fmt.Errorf("wallet id is required")
	}
	if input.Name == "" {
		return CreateBalanceOutput{}, fmt.Errorf("balance name is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateBalanceHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateBalanceOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateBalanceOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateBalanceOutput{}, err
	}
	output.APIVersion = selected
	if output.WalletID == "" {
		output.WalletID = input.WalletID
	}
	return output, nil
}

func (s ListBalancesService) Run(ctx context.Context, input ListBalancesInput) (ListBalancesOutput, error) {
	if input.WalletID == "" {
		return ListBalancesOutput{}, fmt.Errorf("wallet id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListBalancesHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListBalancesOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListBalancesOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListBalancesOutput{}, err
	}
	output.APIVersion = selected
	if output.WalletID == "" {
		output.WalletID = input.WalletID
	}
	return output, nil
}

func (s GetBalanceService) Run(ctx context.Context, input GetBalanceInput) (GetBalanceOutput, error) {
	if input.WalletID == "" {
		return GetBalanceOutput{}, fmt.Errorf("wallet id is required")
	}
	if input.BalanceName == "" {
		return GetBalanceOutput{}, fmt.Errorf("balance name is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetBalanceHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetBalanceOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetBalanceOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetBalanceOutput{}, err
	}
	output.APIVersion = selected
	if output.WalletID == "" {
		output.WalletID = input.WalletID
	}
	return output, nil
}

func (s ListHoldsService) Run(ctx context.Context, input ListHoldsInput) (ListHoldsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListHoldsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListHoldsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListHoldsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListHoldsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetHoldService) Run(ctx context.Context, input GetHoldInput) (GetHoldOutput, error) {
	if input.HoldID == "" {
		return GetHoldOutput{}, fmt.Errorf("hold id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetHoldHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetHoldOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetHoldOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetHoldOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s VoidHoldService) Run(ctx context.Context, input HoldActionInput) (HoldActionOutput, error) {
	return runHoldActionService(ctx, input, s.Handlers, s.Resolve)
}

func (s ConfirmHoldService) Run(ctx context.Context, input HoldActionInput) (HoldActionOutput, error) {
	return runHoldActionService(ctx, input, s.Handlers, s.Resolve)
}

func runHoldActionService(
	ctx context.Context,
	input HoldActionInput,
	handlers []HoldActionHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (HoldActionOutput, error) {
	if input.HoldID == "" {
		return HoldActionOutput{}, fmt.Errorf("hold id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(handlers))
	handlersByVersion := map[capabilities.APIVersion]HoldActionHandler{}
	for _, handler := range handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlersByVersion[handler.APIVersion] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return HoldActionOutput{}, err
	}
	handler, ok := handlersByVersion[selected]
	if !ok {
		return HoldActionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return HoldActionOutput{}, err
	}
	output.APIVersion = selected
	if output.HoldID == "" {
		output.HoldID = input.HoldID
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

func SDKCreditWalletHandlers(sdk *formance.Formance) []WalletMovementHandler {
	return []WalletMovementHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input WalletMovementInput) (WalletMovementOutput, error) {
				_, err := sdk.Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					ID:             input.WalletID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
					CreditWalletRequest: &shared.CreditWalletRequest{
						Amount: shared.Monetary{
							Amount: input.Amount,
							Asset:  input.Asset,
						},
						Balance:  optionalString(input.Balance),
						Metadata: input.Metadata,
					},
				})
				if err != nil {
					return WalletMovementOutput{}, err
				}
				return WalletMovementOutput{WalletID: input.WalletID}, nil
			},
		},
	}
}

func SDKDebitWalletHandlers(sdk *formance.Formance) []WalletMovementHandler {
	return []WalletMovementHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input WalletMovementInput) (WalletMovementOutput, error) {
				response, err := sdk.Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
					ID:             input.WalletID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
					DebitWalletRequest: &shared.DebitWalletRequest{
						Amount: shared.Monetary{
							Amount: input.Amount,
							Asset:  input.Asset,
						},
						Balances:    optionalStringSlice(input.Balance),
						Description: optionalString(input.Description),
						Metadata:    input.Metadata,
						Pending:     &input.Pending,
					},
				})
				if err != nil {
					return WalletMovementOutput{}, err
				}
				output := WalletMovementOutput{WalletID: input.WalletID}
				if response.DebitWalletResponse != nil {
					output.HoldID = response.DebitWalletResponse.Data.ID
				}
				return output, nil
			},
		},
	}
}

func SDKCreateBalanceHandlers(sdk *formance.Formance) []CreateBalanceHandler {
	return []CreateBalanceHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateBalanceInput) (CreateBalanceOutput, error) {
				response, err := sdk.Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
					ID:             input.WalletID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
					CreateBalanceRequest: &shared.CreateBalanceRequest{
						Name:      input.Name,
						Priority:  input.Priority,
						ExpiresAt: input.ExpiresAt,
					},
				})
				if err != nil {
					return CreateBalanceOutput{}, err
				}
				if response.CreateBalanceResponse == nil {
					return CreateBalanceOutput{}, fmt.Errorf("wallets v1 create balance returned no data")
				}
				return CreateBalanceOutput{WalletID: input.WalletID, BalanceName: response.CreateBalanceResponse.Data.Name}, nil
			},
		},
	}
}

func SDKListBalancesHandlers(sdk *formance.Formance) []ListBalancesHandler {
	return []ListBalancesHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListBalancesInput) (ListBalancesOutput, error) {
				response, err := sdk.Wallets.V1.ListBalances(ctx, operations.ListBalancesRequest{ID: input.WalletID})
				if err != nil {
					return ListBalancesOutput{}, err
				}
				if response.ListBalancesResponse == nil {
					return ListBalancesOutput{}, fmt.Errorf("wallets v1 list balances returned no cursor")
				}
				return fromBalancesCursor(input.WalletID, response.ListBalancesResponse.Cursor), nil
			},
		},
	}
}

func SDKGetBalanceHandlers(sdk *formance.Formance) []GetBalanceHandler {
	return []GetBalanceHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetBalanceInput) (GetBalanceOutput, error) {
				response, err := sdk.Wallets.V1.GetBalance(ctx, operations.GetBalanceRequest{ID: input.WalletID, BalanceName: input.BalanceName})
				if err != nil {
					return GetBalanceOutput{}, err
				}
				if response.GetBalanceResponse == nil {
					return GetBalanceOutput{}, fmt.Errorf("wallets v1 get balance returned no data")
				}
				return GetBalanceOutput{WalletID: input.WalletID, Balance: fromBalanceWithAssets(response.GetBalanceResponse.Data)}, nil
			},
		},
	}
}

func SDKListHoldsHandlers(sdk *formance.Formance) []ListHoldsHandler {
	return []ListHoldsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListHoldsInput) (ListHoldsOutput, error) {
				response, err := sdk.Wallets.V1.GetHolds(ctx, operations.GetHoldsRequest{
					PageSize: optionalInt64(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					WalletID: optionalString(input.WalletID),
					Metadata: input.Metadata,
				})
				if err != nil {
					return ListHoldsOutput{}, err
				}
				if response.GetHoldsResponse == nil {
					return ListHoldsOutput{}, fmt.Errorf("wallets v1 list holds returned no cursor")
				}
				return fromHoldsCursor(response.GetHoldsResponse.Cursor), nil
			},
		},
	}
}

func SDKGetHoldHandlers(sdk *formance.Formance) []GetHoldHandler {
	return []GetHoldHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetHoldInput) (GetHoldOutput, error) {
				response, err := sdk.Wallets.V1.GetHold(ctx, operations.GetHoldRequest{HoldID: input.HoldID})
				if err != nil {
					return GetHoldOutput{}, err
				}
				if response.GetHoldResponse == nil {
					return GetHoldOutput{}, fmt.Errorf("wallets v1 get hold returned no data")
				}
				return GetHoldOutput{Hold: fromExpandedHold(response.GetHoldResponse.Data)}, nil
			},
		},
	}
}

func SDKVoidHoldHandlers(sdk *formance.Formance) []HoldActionHandler {
	return []HoldActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input HoldActionInput) (HoldActionOutput, error) {
				_, err := sdk.Wallets.V1.VoidHold(ctx, operations.VoidHoldRequest{
					HoldID:         input.HoldID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
				})
				if err != nil {
					return HoldActionOutput{}, err
				}
				return HoldActionOutput{HoldID: input.HoldID}, nil
			},
		},
	}
}

func SDKConfirmHoldHandlers(sdk *formance.Formance) []HoldActionHandler {
	return []HoldActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input HoldActionInput) (HoldActionOutput, error) {
				_, err := sdk.Wallets.V1.ConfirmHold(ctx, operations.ConfirmHoldRequest{
					HoldID:         input.HoldID,
					IdempotencyKey: optionalString(input.IdempotencyKey),
					ConfirmHoldRequest: &shared.ConfirmHoldRequest{
						Amount: input.Amount,
						Final:  input.Final,
					},
				})
				if err != nil {
					return HoldActionOutput{}, err
				}
				return HoldActionOutput{HoldID: input.HoldID}, nil
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

func fromBalancesCursor(walletID string, cursor shared.ListBalancesResponseCursor) ListBalancesOutput {
	balances := make([]BalanceSummary, 0, len(cursor.Data))
	for _, balance := range cursor.Data {
		balances = append(balances, fromBalance(balance))
	}
	hasMore := false
	if cursor.HasMore != nil {
		hasMore = *cursor.HasMore
	}
	return ListBalancesOutput{
		WalletID: walletID,
		Balances: balances,
		HasMore:  hasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromHoldsCursor(cursor shared.GetHoldsResponseCursor) ListHoldsOutput {
	holds := make([]HoldSummary, 0, len(cursor.Data))
	for _, hold := range cursor.Data {
		holds = append(holds, fromHold(hold))
	}
	hasMore := false
	if cursor.HasMore != nil {
		hasMore = *cursor.HasMore
	}
	return ListHoldsOutput{
		Holds:    holds,
		HasMore:  hasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromHold(hold shared.Hold) HoldSummary {
	return HoldSummary{
		ID:          hold.ID,
		WalletID:    hold.WalletID,
		Asset:       hold.Asset,
		Description: hold.Description,
		Metadata:    hold.Metadata,
	}
}

func fromExpandedHold(hold shared.ExpandedDebitHold) HoldSummary {
	originalAmount := ""
	if hold.OriginalAmount != nil {
		originalAmount = hold.OriginalAmount.String()
	}
	remaining := ""
	if hold.Remaining != nil {
		remaining = hold.Remaining.String()
	}
	return HoldSummary{
		ID:             hold.ID,
		WalletID:       hold.WalletID,
		Asset:          hold.Asset,
		Description:    hold.Description,
		OriginalAmount: originalAmount,
		Remaining:      remaining,
		Metadata:       hold.Metadata,
	}
}

func fromBalance(balance shared.Balance) BalanceSummary {
	priority := ""
	if balance.Priority != nil {
		priority = balance.Priority.String()
	}
	return BalanceSummary{
		Name:      balance.Name,
		Priority:  priority,
		ExpiresAt: balance.ExpiresAt,
	}
}

func fromBalanceWithAssets(balance shared.BalanceWithAssets) BalanceSummary {
	priority := ""
	if balance.Priority != nil {
		priority = balance.Priority.String()
	}
	return BalanceSummary{
		Name:      balance.Name,
		Priority:  priority,
		ExpiresAt: balance.ExpiresAt,
		Assets:    balance.Assets,
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

func optionalStringSlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}
