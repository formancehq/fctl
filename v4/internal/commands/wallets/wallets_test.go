package wallets

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestCreateWalletServiceSelectsResolvedHandler(t *testing.T) {
	service := CreateWalletService{
		Handlers: []CreateWalletHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input CreateWalletInput) (CreateWalletOutput, error) {
					if input.Name != "Main" || input.Metadata["env"] != "dev" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CreateWalletOutput{WalletID: "wallet_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), CreateWalletInput{Name: "Main", Metadata: map[string]string{"env": "dev"}})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.WalletID != "wallet_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateWalletServiceRequiresName(t *testing.T) {
	service := CreateWalletService{
		Handlers: []CreateWalletHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), CreateWalletInput{}); err == nil {
		t.Fatal("expected wallet name validation error")
	}
}

func TestListWalletsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListWalletsService{
		Handlers: []ListWalletsHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input ListWalletsInput) (ListWalletsOutput, error) {
					if input.PageSize != 10 || input.Cursor != "cursor" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListWalletsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), ListWalletsInput{PageSize: 10, Cursor: "cursor"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetWalletServiceRequiresID(t *testing.T) {
	service := GetWalletService{
		Handlers: []GetWalletHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetWalletInput{}); err == nil {
		t.Fatal("expected wallet id validation error")
	}
}

func TestUpdateWalletServiceRequiresMetadata(t *testing.T) {
	service := UpdateWalletService{
		Handlers: []UpdateWalletHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), UpdateWalletInput{WalletID: "wallet_1"}); err == nil {
		t.Fatal("expected metadata validation error")
	}
}

func TestCreditWalletServiceRequiresExplicitWalletID(t *testing.T) {
	service := CreditWalletService{
		Handlers: []WalletMovementHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), WalletMovementInput{Amount: big.NewInt(100), Asset: "USD/2"}); err == nil {
		t.Fatal("expected wallet id validation error")
	}
}

func TestDebitWalletServiceSelectsResolvedHandler(t *testing.T) {
	service := DebitWalletService{
		Handlers: []WalletMovementHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input WalletMovementInput) (WalletMovementOutput, error) {
					if input.WalletID != "wallet_1" || input.Amount.String() != "100" || input.Asset != "USD/2" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return WalletMovementOutput{WalletID: input.WalletID, HoldID: "hold_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), WalletMovementInput{WalletID: "wallet_1", Amount: big.NewInt(100), Asset: "USD/2"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.HoldID != "hold_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateBalanceServiceSelectsResolvedHandler(t *testing.T) {
	service := CreateBalanceService{
		Handlers: []CreateBalanceHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input CreateBalanceInput) (CreateBalanceOutput, error) {
					if input.WalletID != "wallet_1" || input.Name != "main" || input.Priority.String() != "10" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CreateBalanceOutput{BalanceName: input.Name}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), CreateBalanceInput{WalletID: "wallet_1", Name: "main", Priority: big.NewInt(10)})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.WalletID != "wallet_1" || output.BalanceName != "main" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateBalanceServiceRequiresExplicitWalletID(t *testing.T) {
	service := CreateBalanceService{
		Handlers: []CreateBalanceHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), CreateBalanceInput{Name: "main"}); err == nil {
		t.Fatal("expected wallet id validation error")
	}
}

func TestListBalancesServiceRequiresExplicitWalletID(t *testing.T) {
	service := ListBalancesService{
		Handlers: []ListBalancesHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), ListBalancesInput{}); err == nil {
		t.Fatal("expected wallet id validation error")
	}
}

func TestGetBalanceServiceRequiresBalanceName(t *testing.T) {
	service := GetBalanceService{
		Handlers: []GetBalanceHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetBalanceInput{WalletID: "wallet_1"}); err == nil {
		t.Fatal("expected balance name validation error")
	}
}

func TestListHoldsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListHoldsService{
		Handlers: []ListHoldsHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input ListHoldsInput) (ListHoldsOutput, error) {
					if input.PageSize != 10 || input.WalletID != "wallet_1" || input.Metadata["env"] != "dev" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListHoldsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), ListHoldsInput{PageSize: 10, WalletID: "wallet_1", Metadata: map[string]string{"env": "dev"}})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetHoldServiceRequiresHoldID(t *testing.T) {
	service := GetHoldService{
		Handlers: []GetHoldHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetHoldInput{}); err == nil {
		t.Fatal("expected hold id validation error")
	}
}

func TestConfirmHoldServiceSelectsResolvedHandler(t *testing.T) {
	service := ConfirmHoldService{
		Handlers: []HoldActionHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input HoldActionInput) (HoldActionOutput, error) {
					if input.HoldID != "hold_1" || input.Amount.String() != "100" || input.Final == nil || !*input.Final {
						t.Fatalf("unexpected input: %#v", input)
					}
					return HoldActionOutput{HoldID: input.HoldID}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	final := true
	output, err := service.Run(context.Background(), HoldActionInput{HoldID: "hold_1", Amount: big.NewInt(100), Final: &final})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.HoldID != "hold_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestVoidHoldServiceRequiresHoldID(t *testing.T) {
	service := VoidHoldService{
		Handlers: []HoldActionHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), HoldActionInput{}); err == nil {
		t.Fatal("expected hold id validation error")
	}
}

func TestListTransactionsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListTransactionsService{
		Handlers: []ListTransactionsHandler{
			{
				APIVersion: "v1",
				Run: func(_ context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
					if input.PageSize != 10 || input.WalletID != "wallet_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListTransactionsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background(), ListTransactionsInput{PageSize: 10, WalletID: "wallet_1"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func assertAPIVersions(t *testing.T, got []capabilities.APIVersion, want []capabilities.APIVersion) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected versions %v, got %v", want, got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("expected versions %v, got %v", want, got)
		}
	}
}
