package payments

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListBankAccountsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListBankAccountsService{
		Handlers: []ListBankAccountsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListBankAccountsInput) (ListBankAccountsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListBankAccountsOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ListBankAccountsInput) (ListBankAccountsOutput, error) {
					if input.PageSize != 10 {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListBankAccountsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListBankAccountsInput{PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateBankAccountServiceSelectsResolvedHandler(t *testing.T) {
	service := CreateBankAccountService{
		Handlers: []CreateBankAccountHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, CreateBankAccountInput) (CreateBankAccountOutput, error) {
					t.Fatal("v1 handler should not run")
					return CreateBankAccountOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input CreateBankAccountInput) (CreateBankAccountOutput, error) {
					if input.Name != "Main" || input.Country != "FR" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CreateBankAccountOutput{BankAccountID: "ba_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), CreateBankAccountInput{Name: "Main", Country: "FR"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.BankAccountID != "ba_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateBankAccountServiceRequiresName(t *testing.T) {
	service := CreateBankAccountService{
		Handlers: []CreateBankAccountHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), CreateBankAccountInput{}); err == nil {
		t.Fatal("expected bank account name validation error")
	}
}

func TestForwardBankAccountServiceSelectsResolvedHandler(t *testing.T) {
	service := ForwardBankAccountService{
		Handlers: []ForwardBankAccountHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ForwardBankAccountInput) (ForwardBankAccountOutput, error) {
					t.Fatal("v1 handler should not run")
					return ForwardBankAccountOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ForwardBankAccountInput) (ForwardBankAccountOutput, error) {
					if input.BankAccountID != "ba_1" || input.ConnectorID != "conn_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ForwardBankAccountOutput{BankAccountID: input.BankAccountID, ConnectorID: input.ConnectorID, TaskID: "task_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ForwardBankAccountInput{BankAccountID: "ba_1", ConnectorID: "conn_1"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.TaskID != "task_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestSetBankAccountMetadataServiceRequiresMetadata(t *testing.T) {
	service := SetBankAccountMetadataService{
		Handlers: []SetBankAccountMetadataHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), SetBankAccountMetadataInput{BankAccountID: "ba_1"}); err == nil {
		t.Fatal("expected bank account metadata validation error")
	}
}

func TestGetBankAccountServiceRequiresID(t *testing.T) {
	service := GetBankAccountService{
		Handlers: []GetBankAccountHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetBankAccountInput{}); err == nil {
		t.Fatal("expected bank account id validation error")
	}
}
