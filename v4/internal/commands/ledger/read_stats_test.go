package ledger

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestReadStatsServiceSelectsResolvedHandler(t *testing.T) {
	service := ReadStatsService{
		Handlers: []ReadStatsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ReadStatsInput) (ReadStatsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ReadStatsOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ReadStatsInput) (ReadStatsOutput, error) {
					if input.Ledger != "default" {
						t.Fatalf("unexpected ledger %q", input.Ledger)
					}
					return ReadStatsOutput{Accounts: 2, Transactions: "42"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ReadStatsInput{Ledger: "default"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Accounts != 2 || output.Transactions != "42" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestReadStatsServiceReturnsResolverError(t *testing.T) {
	expected := errors.New("unsupported")
	service := ReadStatsService{
		Handlers: []ReadStatsHandler{{APIVersion: "v1", Run: nil}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			return "", expected
		},
	}

	_, err := service.Run(context.Background(), ReadStatsInput{Ledger: "default"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestReadStatsServiceRequiresLedger(t *testing.T) {
	service := ReadStatsService{}
	_, err := service.Run(context.Background(), ReadStatsInput{})
	if err == nil || err.Error() != "ledger is required" {
		t.Fatalf("expected ledger required error, got %v", err)
	}
}

func TestStatsMapping(t *testing.T) {
	v1 := fromV1Stats(shared.Stats{Accounts: 2, Transactions: 42})
	if v1.Accounts != 2 || v1.Transactions != "42" {
		t.Fatalf("unexpected v1 stats: %#v", v1)
	}

	v2 := fromV2Stats(shared.V2Stats{Accounts: 3, Transactions: big.NewInt(420000000000)})
	if v2.Accounts != 3 || v2.Transactions != "420000000000" {
		t.Fatalf("unexpected v2 stats: %#v", v2)
	}
}
