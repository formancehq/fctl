package ledger

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestListTransactionsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListTransactionsService{
		Handlers: []ListTransactionsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListTransactionsInput) (ListTransactionsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListTransactionsOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListTransactionsInput) (ListTransactionsOutput, error) {
					if input.Ledger != "default" {
						t.Fatalf("unexpected ledger %q", input.Ledger)
					}
					return ListTransactionsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListTransactionsInput{Ledger: "default", PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestListTransactionsServiceReturnsResolverError(t *testing.T) {
	expected := errors.New("unsupported")
	service := ListTransactionsService{
		Handlers: []ListTransactionsHandler{{APIVersion: "v3", Run: nil}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			return "", expected
		},
	}

	_, err := service.Run(context.Background(), ListTransactionsInput{Ledger: "default"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestCountTransactionsServiceSelectsResolvedHandler(t *testing.T) {
	service := CountTransactionsService{
		Handlers: []CountTransactionsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, CountTransactionsInput) (CountTransactionsOutput, error) {
					t.Fatal("v1 handler should not run")
					return CountTransactionsOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input CountTransactionsInput) (CountTransactionsOutput, error) {
					if input.Ledger != "default" || input.Source != "world" || input.Destination != "users:123" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CountTransactionsOutput{Count: 42}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), CountTransactionsInput{
		Ledger:      "default",
		Source:      "world",
		Destination: "users:123",
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Count != 42 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCountTransactionsRequestMapsCanonicalFilters(t *testing.T) {
	input := CountTransactionsInput{
		Ledger:      "default",
		Account:     "users:123",
		Source:      "world",
		Destination: "users:123",
		Reference:   "ref",
	}

	v1 := toV1CountTransactionsRequest(input)
	if v1.Ledger != "default" || *v1.Account != "users:123" || *v1.Source != "world" ||
		*v1.Destination != "users:123" || *v1.Reference != "ref" {
		t.Fatalf("unexpected v1 request: %#v", v1)
	}

	v2 := toV2CountTransactionsRequest(input)
	if v2.Ledger != "default" || v2.Query["account"] != "users:123" || v2.Query["source"] != "world" ||
		v2.Query["destination"] != "users:123" || v2.Query["reference"] != "ref" {
		t.Fatalf("unexpected v2 request: %#v", v2)
	}
}

func TestCountHeader(t *testing.T) {
	count, err := countHeader(map[string][]string{"Count": {"42"}})
	if err != nil {
		t.Fatalf("parse count header: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected count 42, got %d", count)
	}

	if _, err := countHeader(map[string][]string{}); err == nil {
		t.Fatal("expected missing Count header error")
	}
	if _, err := countHeader(map[string][]string{"Count": {"wat"}}); err == nil {
		t.Fatal("expected invalid Count header error")
	}
}

func TestSendTransactionServiceSelectsResolvedHandler(t *testing.T) {
	service := SendTransactionService{
		Handlers: []SendTransactionHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, SendTransactionInput) (SendTransactionOutput, error) {
					t.Fatal("v1 handler should not run")
					return SendTransactionOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input SendTransactionInput) (SendTransactionOutput, error) {
					if input.Ledger != "default" || input.Source != "world" || input.Destination != "users:123" ||
						input.Amount != "100" || input.Asset != "USD/2" || input.Metadata["foo"] != "bar" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return SendTransactionOutput{Transaction: TransactionSummary{ID: "42"}}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), SendTransactionInput{
		Ledger:      "default",
		Source:      "world",
		Destination: "users:123",
		Amount:      "100",
		Asset:       "USD/2",
		Metadata:    map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Transaction.ID != "42" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestSendTransactionServiceRequiresInput(t *testing.T) {
	service := SendTransactionService{
		Handlers: []SendTransactionHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), SendTransactionInput{}); err == nil {
		t.Fatal("expected ledger validation error")
	}
	if _, err := service.Run(context.Background(), SendTransactionInput{Ledger: "default"}); err == nil {
		t.Fatal("expected source validation error")
	}
}

func TestParseAmount(t *testing.T) {
	amount, err := parseAmount("100")
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}
	if amount.String() != "100" {
		t.Fatalf("expected amount 100, got %s", amount)
	}

	if _, err := parseAmount("not-an-int"); err == nil {
		t.Fatal("expected invalid amount error")
	}
}

func TestToV2ListTransactionsRequestMapsCanonicalFilters(t *testing.T) {
	request := toV2ListTransactionsRequest(ListTransactionsInput{
		Ledger:      "default",
		PageSize:    10,
		Account:     "users:123",
		Source:      "world",
		Destination: "users:123",
		Reference:   "ref",
	})

	if request.Ledger != "default" || *request.PageSize != 10 {
		t.Fatalf("unexpected base request: %#v", request)
	}
	if request.Query["account"] != "users:123" || request.Query["source"] != "world" ||
		request.Query["destination"] != "users:123" || request.Query["reference"] != "ref" {
		t.Fatalf("unexpected query mapping: %#v", request.Query)
	}
}

func TestGetTransactionServiceSelectsResolvedHandler(t *testing.T) {
	service := GetTransactionService{
		Handlers: []GetTransactionHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, GetTransactionInput) (GetTransactionOutput, error) {
					t.Fatal("v1 handler should not run")
					return GetTransactionOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input GetTransactionInput) (GetTransactionOutput, error) {
					if input.Ledger != "default" || input.TransactionID != "42" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return GetTransactionOutput{Transaction: TransactionSummary{ID: input.TransactionID}}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), GetTransactionInput{Ledger: "default", TransactionID: "42"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Transaction.ID != "42" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestRevertTransactionServiceSelectsResolvedHandler(t *testing.T) {
	service := RevertTransactionService{
		Handlers: []RevertTransactionHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, RevertTransactionInput) (RevertTransactionOutput, error) {
					t.Fatal("v1 handler should not run")
					return RevertTransactionOutput{}, nil
				},
			},
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input RevertTransactionInput) (RevertTransactionOutput, error) {
					if input.Ledger != "default" || input.TransactionID != "42" || !input.AtEffectiveDate || !input.Force {
						t.Fatalf("unexpected input: %#v", input)
					}
					return RevertTransactionOutput{Transaction: TransactionSummary{ID: input.TransactionID}}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), RevertTransactionInput{
		Ledger:          "default",
		TransactionID:   "42",
		AtEffectiveDate: true,
		Force:           true,
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Transaction.ID != "42" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestRevertTransactionServiceRequiresInput(t *testing.T) {
	service := RevertTransactionService{
		Handlers: []RevertTransactionHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), RevertTransactionInput{TransactionID: "42"}); err == nil {
		t.Fatal("expected ledger validation error")
	}
	if _, err := service.Run(context.Background(), RevertTransactionInput{Ledger: "default"}); err == nil {
		t.Fatal("expected transaction id validation error")
	}
}

func TestTransactionMapping(t *testing.T) {
	timestamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	reference := "ref"

	v1 := fromV1Transaction(shared.Transaction{
		Txid:      big.NewInt(42),
		Reference: &reference,
		Timestamp: timestamp,
		Metadata:  map[string]any{"foo": "bar"},
	})
	if v1.ID != "42" || *v1.Reference != "ref" || !v1.Timestamp.Equal(timestamp) || v1.Metadata["foo"] != "bar" {
		t.Fatalf("unexpected v1 transaction: %#v", v1)
	}

	v2 := fromV2Transaction(shared.V2Transaction{
		ID:        big.NewInt(43),
		Reference: &reference,
		Timestamp: timestamp,
		Metadata:  map[string]string{"foo": "bar"},
	})
	if v2.ID != "43" || *v2.Reference != "ref" || !v2.Timestamp.Equal(timestamp) || v2.Metadata["foo"] != "bar" {
		t.Fatalf("unexpected v2 transaction: %#v", v2)
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
