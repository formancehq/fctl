package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestAddAccountMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := AddAccountMetadataService{
		Handlers: []AddAccountMetadataHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, AddAccountMetadataInput) (AddAccountMetadataOutput, error) {
					t.Fatal("v1 handler should not run")
					return AddAccountMetadataOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input AddAccountMetadataInput) (AddAccountMetadataOutput, error) {
					if input.Ledger != "default" || input.Account != "users:123" || input.Metadata["foo"] != "bar" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return AddAccountMetadataOutput{Updated: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), AddAccountMetadataInput{
		Ledger:   "default",
		Account:  "users:123",
		Metadata: map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Updated {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestAddAccountMetadataServiceRequiresInput(t *testing.T) {
	service := AddAccountMetadataService{
		Handlers: []AddAccountMetadataHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), AddAccountMetadataInput{Account: "users:123", Metadata: map[string]string{"foo": "bar"}}); err == nil {
		t.Fatal("expected ledger validation error")
	}
	if _, err := service.Run(context.Background(), AddAccountMetadataInput{Ledger: "default", Metadata: map[string]string{"foo": "bar"}}); err == nil {
		t.Fatal("expected account validation error")
	}
	if _, err := service.Run(context.Background(), AddAccountMetadataInput{Ledger: "default", Account: "users:123"}); err == nil {
		t.Fatal("expected metadata validation error")
	}
}

func TestDeleteAccountMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := DeleteAccountMetadataService{
		Handlers: []DeleteAccountMetadataHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input DeleteAccountMetadataInput) (DeleteAccountMetadataOutput, error) {
					if input.Ledger != "default" || input.Account != "users:123" || input.Key != "foo" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return DeleteAccountMetadataOutput{Deleted: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), DeleteAccountMetadataInput{
		Ledger:  "default",
		Account: "users:123",
		Key:     "foo",
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Deleted {
		t.Fatalf("unexpected output: %#v", output)
	}
}
