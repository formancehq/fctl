package payments

import (
	"context"
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
