package payments

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

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

func TestCreatePaymentServiceSelectsResolvedHandler(t *testing.T) {
	service := CreatePaymentService{
		Handlers: []CreatePaymentHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, CreatePaymentInput) (CreatePaymentOutput, error) {
					t.Fatal("v1 handler should not run")
					return CreatePaymentOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input CreatePaymentInput) (CreatePaymentOutput, error) {
					if input.V3.Amount.String() != "100" || input.V3.Asset != "USD/2" || input.V3.ConnectorID != "conn_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CreatePaymentOutput{PaymentID: "pay_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), CreatePaymentInput{
		V3: newTestCreatePaymentRequest(),
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PaymentID != "pay_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreatePaymentServiceRequiresAmount(t *testing.T) {
	service := CreatePaymentService{
		Handlers: []CreatePaymentHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	request := newTestCreatePaymentRequest()
	request.Amount = nil
	if _, err := service.Run(context.Background(), CreatePaymentInput{V3: request}); err == nil {
		t.Fatal("expected payment amount validation error")
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

func newTestCreatePaymentRequest() shared.V3CreatePaymentRequest {
	return shared.V3CreatePaymentRequest{
		Amount:        big.NewInt(100),
		InitialAmount: big.NewInt(100),
		Asset:         "USD/2",
		ConnectorID:   "conn_1",
		Type:          shared.V3PaymentTypeEnumPayout,
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
