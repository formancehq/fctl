package payments

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListPaymentsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListPaymentsService{
		Handlers: []ListPaymentsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListPaymentsInput) (ListPaymentsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListPaymentsOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ListPaymentsInput) (ListPaymentsOutput, error) {
					if input.PageSize != 10 {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListPaymentsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListPaymentsInput{PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetPaymentServiceRequiresID(t *testing.T) {
	service := GetPaymentService{
		Handlers: []GetPaymentHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetPaymentInput{}); err == nil {
		t.Fatal("expected payment id validation error")
	}
}

func TestSetPaymentMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := SetPaymentMetadataService{
		Handlers: []SetPaymentMetadataHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, SetPaymentMetadataInput) (SetPaymentMetadataOutput, error) {
					t.Fatal("v1 handler should not run")
					return SetPaymentMetadataOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input SetPaymentMetadataInput) (SetPaymentMetadataOutput, error) {
					if input.PaymentID != "pay_1" || input.Metadata["env"] != "dev" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return SetPaymentMetadataOutput{PaymentID: input.PaymentID}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), SetPaymentMetadataInput{PaymentID: "pay_1", Metadata: map[string]string{"env": "dev"}})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PaymentID != "pay_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestSetPaymentMetadataServiceRequiresMetadata(t *testing.T) {
	service := SetPaymentMetadataService{
		Handlers: []SetPaymentMetadataHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), SetPaymentMetadataInput{PaymentID: "pay_1"}); err == nil {
		t.Fatal("expected payment metadata validation error")
	}
}
