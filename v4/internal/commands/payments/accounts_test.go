package payments

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
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
				APIVersion: "v3",
				Run: func(_ context.Context, input ListAccountsInput) (ListAccountsOutput, error) {
					if input.PageSize != 10 || input.Cursor != "cursor" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListAccountsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListAccountsInput{PageSize: 10, Cursor: "cursor"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetAccountServiceRequiresAccountID(t *testing.T) {
	service := GetAccountService{
		Handlers: []GetAccountHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetAccountInput{}); err == nil {
		t.Fatal("expected account id validation error")
	}
}

func assertAPIVersions(t *testing.T, got []capabilities.APIVersion, want []capabilities.APIVersion) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected versions %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected versions %v, got %v", want, got)
		}
	}
}
