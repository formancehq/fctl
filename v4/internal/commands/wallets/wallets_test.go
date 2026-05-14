package wallets

import (
	"context"
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
