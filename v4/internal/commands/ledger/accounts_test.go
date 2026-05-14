package ledger

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestListAccountsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListAccountsService{
		Handlers: []ListAccountsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListAccountsInput) (ListAccountsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListAccountsOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
					if input.Ledger != "default" || input.Account != "users:123" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListAccountsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListAccountsInput{Ledger: "default", Account: "users:123", PageSize: 15})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 15 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestToListAccountsRequestsMapCanonicalInput(t *testing.T) {
	input := ListAccountsInput{
		Ledger:   "default",
		PageSize: 15,
		Cursor:   "cursor",
		Account:  "users:123",
		Metadata: map[string]string{"tier": "gold"},
	}

	v1 := toV1ListAccountsRequest(input)
	if v1.Ledger != "default" || *v1.PageSize != 15 || *v1.Cursor != "cursor" || *v1.Address != "users:123" {
		t.Fatalf("unexpected v1 request: %#v", v1)
	}
	if v1.Metadata["tier"] != "gold" {
		t.Fatalf("unexpected v1 metadata: %#v", v1.Metadata)
	}

	v2 := toV2ListAccountsRequest(input)
	if v2.Ledger != "default" || *v2.PageSize != 15 || *v2.Cursor != "cursor" {
		t.Fatalf("unexpected v2 request: %#v", v2)
	}
	andParts, ok := v2.Query["$and"].([]map[string]any)
	if !ok || len(andParts) != 2 {
		t.Fatalf("unexpected v2 query: %#v", v2.Query)
	}
}

func TestGetAccountServiceRequiresInputs(t *testing.T) {
	service := GetAccountService{}
	if _, err := service.Run(context.Background(), GetAccountInput{}); err == nil || err.Error() != "ledger is required" {
		t.Fatalf("expected ledger required error, got %v", err)
	}
	if _, err := service.Run(context.Background(), GetAccountInput{Ledger: "default"}); err == nil || err.Error() != "account is required" {
		t.Fatalf("expected account required error, got %v", err)
	}
}

func TestAccountMapping(t *testing.T) {
	v1 := fromV1AccountDetail(shared.AccountWithVolumesAndBalances{
		Address:  "users:123",
		Metadata: map[string]any{"tier": "gold"},
		Volumes: map[string]shared.Volume{
			"USD/2": {Input: big.NewInt(100), Output: big.NewInt(40), Balance: big.NewInt(60)},
		},
	})
	if v1.Address != "users:123" || v1.Metadata["tier"] != "gold" || v1.Volumes["USD/2"].Balance != "60" {
		t.Fatalf("unexpected v1 detail: %#v", v1)
	}

	v2 := fromV2AccountDetail(shared.V2Account{
		Address:  "users:123",
		Metadata: map[string]string{"tier": "gold"},
		Volumes: map[string]shared.V2Volume{
			"USD/2": {Input: big.NewInt(100), Output: big.NewInt(40), Balance: big.NewInt(60)},
		},
	})
	if v2.Address != "users:123" || v2.Metadata["tier"] != "gold" || v2.Volumes["USD/2"].Input != "100" {
		t.Fatalf("unexpected v2 detail: %#v", v2)
	}
}
