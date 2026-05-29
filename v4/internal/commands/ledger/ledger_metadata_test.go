package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestUpdateLedgerMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := UpdateLedgerMetadataService{
		Handlers: []UpdateLedgerMetadataHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input UpdateLedgerMetadataInput) (UpdateLedgerMetadataOutput, error) {
					if input.Ledger != "default" || input.Metadata["tier"] != "gold" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return UpdateLedgerMetadataOutput{Updated: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), UpdateLedgerMetadataInput{
		Ledger:   "default",
		Metadata: map[string]string{"tier": "gold"},
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Updated {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestUpdateLedgerMetadataServiceRequiresInput(t *testing.T) {
	service := UpdateLedgerMetadataService{
		Handlers: []UpdateLedgerMetadataHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), UpdateLedgerMetadataInput{Metadata: map[string]string{"tier": "gold"}}); err == nil {
		t.Fatal("expected ledger validation error")
	}
	if _, err := service.Run(context.Background(), UpdateLedgerMetadataInput{Ledger: "default"}); err == nil {
		t.Fatal("expected metadata validation error")
	}
}

func TestDeleteLedgerMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := DeleteLedgerMetadataService{
		Handlers: []DeleteLedgerMetadataHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input DeleteLedgerMetadataInput) (DeleteLedgerMetadataOutput, error) {
					if input.Ledger != "default" || input.Key != "tier" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return DeleteLedgerMetadataOutput{Deleted: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), DeleteLedgerMetadataInput{
		Ledger: "default",
		Key:    "tier",
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Deleted {
		t.Fatalf("unexpected output: %#v", output)
	}
}
