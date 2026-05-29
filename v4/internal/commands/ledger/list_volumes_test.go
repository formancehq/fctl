package ledger

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestListVolumesServiceSelectsResolvedHandler(t *testing.T) {
	service := ListVolumesService{
		Handlers: []ListVolumesHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListVolumesInput) (ListVolumesOutput, error) {
					if input.Ledger != "default" || input.Account != "users:123" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListVolumesOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListVolumesInput{Ledger: "default", Account: "users:123", PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestToV2ListVolumesRequestMapsCanonicalInput(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	request := toV2ListVolumesRequest(ListVolumesInput{
		Ledger:           "default",
		PageSize:         10,
		Cursor:           "cursor",
		Account:          "users:123",
		Metadata:         map[string]string{"tier": "gold"},
		StartTime:        &start,
		EndTime:          &end,
		UseInsertionDate: true,
		GroupBy:          1,
		Sort:             "account:asc",
	})

	if request.Ledger != "default" || *request.PageSize != 10 || *request.Cursor != "cursor" {
		t.Fatalf("unexpected request: %#v", request)
	}
	if request.StartTime == nil || !request.StartTime.Equal(start) || request.EndTime == nil || !request.EndTime.Equal(end) {
		t.Fatalf("unexpected times: %#v", request)
	}
	if request.InsertionDate == nil || !*request.InsertionDate || request.GroupBy == nil || *request.GroupBy != 1 {
		t.Fatalf("unexpected grouping fields: %#v", request)
	}
	andParts, ok := request.Query["$and"].([]map[string]any)
	if !ok || len(andParts) != 2 {
		t.Fatalf("unexpected query: %#v", request.Query)
	}
}

func TestFromV2ListVolumesMapsCursor(t *testing.T) {
	output := fromV2ListVolumes(shared.V2VolumesWithBalanceCursorResponseCursor{
		Data: []shared.V2VolumesWithBalance{
			{
				Account: "users:123",
				Asset:   "USD/2",
				Input:   big.NewInt(100),
				Output:  big.NewInt(40),
				Balance: big.NewInt(60),
			},
		},
		PageSize: 10,
	})

	if len(output.Volumes) != 1 {
		t.Fatalf("expected one volume, got %#v", output.Volumes)
	}
	if output.Volumes[0].Account != "users:123" || output.Volumes[0].Balance != "60" {
		t.Fatalf("unexpected volume: %#v", output.Volumes[0])
	}
}
