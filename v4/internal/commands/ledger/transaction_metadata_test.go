package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestAddTransactionMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := AddTransactionMetadataService{
		Handlers: []AddTransactionMetadataHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, AddTransactionMetadataInput) (AddTransactionMetadataOutput, error) {
					t.Fatal("v1 handler should not run")
					return AddTransactionMetadataOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input AddTransactionMetadataInput) (AddTransactionMetadataOutput, error) {
					if input.Ledger != "default" || input.TransactionID != "42" || input.Metadata["foo"] != "bar" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return AddTransactionMetadataOutput{Updated: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), AddTransactionMetadataInput{
		Ledger:        "default",
		TransactionID: "42",
		Metadata:      map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Updated {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestAddTransactionMetadataServiceRequiresInput(t *testing.T) {
	service := AddTransactionMetadataService{
		Handlers: []AddTransactionMetadataHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), AddTransactionMetadataInput{TransactionID: "42", Metadata: map[string]string{"foo": "bar"}}); err == nil {
		t.Fatal("expected ledger validation error")
	}
	if _, err := service.Run(context.Background(), AddTransactionMetadataInput{Ledger: "default", Metadata: map[string]string{"foo": "bar"}}); err == nil {
		t.Fatal("expected transaction id validation error")
	}
	if _, err := service.Run(context.Background(), AddTransactionMetadataInput{Ledger: "default", TransactionID: "42"}); err == nil {
		t.Fatal("expected metadata validation error")
	}
}

func TestDeleteTransactionMetadataServiceSelectsResolvedHandler(t *testing.T) {
	service := DeleteTransactionMetadataService{
		Handlers: []DeleteTransactionMetadataHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input DeleteTransactionMetadataInput) (DeleteTransactionMetadataOutput, error) {
					if input.Ledger != "default" || input.TransactionID != "42" || input.Key != "foo" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return DeleteTransactionMetadataOutput{Deleted: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), DeleteTransactionMetadataInput{
		Ledger:        "default",
		TransactionID: "42",
		Key:           "foo",
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || !output.Deleted {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestParseTransactionID(t *testing.T) {
	txid, err := parseTransactionID("42")
	if err != nil {
		t.Fatalf("parse transaction id: %v", err)
	}
	if txid.String() != "42" {
		t.Fatalf("expected transaction id 42, got %s", txid)
	}

	if _, err := parseTransactionID("not-an-int"); err == nil {
		t.Fatal("expected invalid transaction id error")
	}
}
