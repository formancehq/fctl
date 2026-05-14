package payments

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListTransferInitiationsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListTransferInitiationsService{
		Handlers: []ListTransferInitiationsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListTransferInitiationsInput) (ListTransferInitiationsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListTransferInitiationsOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ListTransferInitiationsInput) (ListTransferInitiationsOutput, error) {
					if input.PageSize != 10 {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListTransferInitiationsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListTransferInitiationsInput{PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetTransferInitiationServiceRequiresID(t *testing.T) {
	service := GetTransferInitiationService{
		Handlers: []GetTransferInitiationHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetTransferInitiationInput{}); err == nil {
		t.Fatal("expected transfer initiation id validation error")
	}
}

func TestTransferInitiationActionServiceSelectsResolvedHandler(t *testing.T) {
	service := TransferInitiationActionService{
		Handlers: []TransferInitiationActionHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
					t.Fatal("v1 handler should not run")
					return TransferInitiationActionOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input TransferInitiationActionInput) (TransferInitiationActionOutput, error) {
					if input.TransferInitiationID != "ti_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return TransferInitiationActionOutput{TransferInitiationID: input.TransferInitiationID}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), TransferInitiationActionInput{TransferInitiationID: "ti_1"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.TransferInitiationID != "ti_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestTransferInitiationActionServiceRequiresID(t *testing.T) {
	service := TransferInitiationActionService{
		Handlers: []TransferInitiationActionHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), TransferInitiationActionInput{}); err == nil {
		t.Fatal("expected transfer initiation id validation error")
	}
}

func TestUpdateTransferInitiationStatusServiceRequiresStatus(t *testing.T) {
	service := UpdateTransferInitiationStatusService{
		Handlers: []UpdateTransferInitiationStatusHandler{{APIVersion: "v1"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), UpdateTransferInitiationStatusInput{TransferInitiationID: "ti_1"}); err == nil {
		t.Fatal("expected transfer initiation status validation error")
	}
}

func TestReverseTransferInitiationServiceSelectsResolvedHandler(t *testing.T) {
	service := ReverseTransferInitiationService{
		Handlers: []ReverseTransferInitiationHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ReverseTransferInitiationInput) (ReverseTransferInitiationOutput, error) {
					t.Fatal("v1 handler should not run")
					return ReverseTransferInitiationOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ReverseTransferInitiationInput) (ReverseTransferInitiationOutput, error) {
					if input.TransferInitiationID != "ti_1" || input.Amount.String() != "100" || input.Asset != "USD/2" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ReverseTransferInitiationOutput{TransferInitiationID: input.TransferInitiationID, TaskID: "task_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ReverseTransferInitiationInput{
		TransferInitiationID: "ti_1",
		Amount:               big.NewInt(100),
		Asset:                "USD/2",
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.TaskID != "task_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestReverseTransferInitiationServiceRequiresAmount(t *testing.T) {
	service := ReverseTransferInitiationService{
		Handlers: []ReverseTransferInitiationHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), ReverseTransferInitiationInput{TransferInitiationID: "ti_1", Asset: "USD/2"}); err == nil {
		t.Fatal("expected reverse transfer initiation amount validation error")
	}
}
