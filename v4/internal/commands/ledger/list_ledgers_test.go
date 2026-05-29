package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListLedgersServiceSelectsResolvedHandler(t *testing.T) {
	service := ListLedgersService{
		Handlers: []ListLedgersHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListLedgersInput) (ListLedgersOutput, error) {
					if input.PageSize != 10 || !input.IncludeDeleted {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListLedgersOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListLedgersInput{PageSize: 10, IncludeDeleted: true})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateLedgerServiceSelectsResolvedHandler(t *testing.T) {
	service := CreateLedgerService{
		Handlers: []CreateLedgerHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input CreateLedgerInput) (CreateLedgerOutput, error) {
					if input.Name != "default" || input.Bucket != "bucket" || input.Features["hash"] != "true" || input.Metadata["tier"] != "gold" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return CreateLedgerOutput{Name: input.Name, Created: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), CreateLedgerInput{
		Name:     "default",
		Bucket:   "bucket",
		Features: map[string]string{"hash": "true"},
		Metadata: map[string]string{"tier": "gold"},
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Name != "default" || !output.Created {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCreateLedgerServiceRequiresName(t *testing.T) {
	service := CreateLedgerService{
		Handlers: []CreateLedgerHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), CreateLedgerInput{}); err == nil {
		t.Fatal("expected ledger name validation error")
	}
}

func TestToV2ListLedgersRequestMapsCanonicalInput(t *testing.T) {
	request := toV2ListLedgersRequest(ListLedgersInput{
		PageSize:       10,
		Cursor:         "cursor",
		IncludeDeleted: true,
		Sort:           "name:asc",
	})

	if request.PageSize == nil || *request.PageSize != 10 {
		t.Fatalf("unexpected page size: %#v", request.PageSize)
	}
	if request.Cursor == nil || *request.Cursor != "cursor" {
		t.Fatalf("unexpected cursor: %#v", request.Cursor)
	}
	if request.IncludeDeleted == nil || !*request.IncludeDeleted {
		t.Fatalf("unexpected include deleted: %#v", request.IncludeDeleted)
	}
	if request.Sort == nil || *request.Sort != "name:asc" {
		t.Fatalf("unexpected sort: %#v", request.Sort)
	}
}

func TestFromV2ListLedgersMapsCursor(t *testing.T) {
	addedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	output := fromV2ListLedgers(shared.V2LedgerListResponseCursor{
		Data: []shared.V2Ledger{
			{
				Name:    "default",
				Bucket:  "bucket",
				AddedAt: addedAt,
			},
		},
		PageSize: 15,
	})

	if len(output.Ledgers) != 1 {
		t.Fatalf("expected one ledger, got %#v", output.Ledgers)
	}
	if output.Ledgers[0].Name != "default" || output.Ledgers[0].Bucket != "bucket" {
		t.Fatalf("unexpected ledger: %#v", output.Ledgers[0])
	}
	if !output.Ledgers[0].AddedAt.Equal(addedAt) {
		t.Fatalf("unexpected addedAt: %s", output.Ledgers[0].AddedAt)
	}
	if output.PageSize != 15 || output.HasMore {
		t.Fatalf("unexpected cursor fields: %#v", output)
	}
}
