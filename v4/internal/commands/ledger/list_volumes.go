package ledger

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const FeatureGetVolumesWithBalances capabilities.Feature = "getVolumesWithBalances"

type ListVolumesInput struct {
	Ledger           string
	PageSize         int64
	Cursor           string
	Account          string
	Metadata         map[string]string
	StartTime        *time.Time
	EndTime          *time.Time
	UseInsertionDate bool
	GroupBy          int64
	Sort             string
}

type ListVolumesOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Volumes    []VolumeBalance         `json:"volumes" yaml:"volumes"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type VolumeBalance struct {
	Account string `json:"account" yaml:"account"`
	Asset   string `json:"asset" yaml:"asset"`
	Input   string `json:"input" yaml:"input"`
	Output  string `json:"output" yaml:"output"`
	Balance string `json:"balance" yaml:"balance"`
}

type ListVolumesHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListVolumesInput) (ListVolumesOutput, error)
}

type ListVolumesService struct {
	Handlers []ListVolumesHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListVolumesService) Run(ctx context.Context, input ListVolumesInput) (ListVolumesOutput, error) {
	if input.Ledger == "" {
		return ListVolumesOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListVolumesHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListVolumesOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListVolumesOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListVolumesOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListVolumesHandlers(sdk *formance.Formance) []ListVolumesHandler {
	return []ListVolumesHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListVolumesInput) (ListVolumesOutput, error) {
				response, err := sdk.Ledger.V2.GetVolumesWithBalances(ctx, toV2ListVolumesRequest(input))
				if err != nil {
					return ListVolumesOutput{}, err
				}
				if response.V2VolumesWithBalanceCursorResponse == nil {
					return ListVolumesOutput{}, fmt.Errorf("ledger v2 list volumes returned no cursor")
				}
				return fromV2ListVolumes(response.V2VolumesWithBalanceCursorResponse.Cursor), nil
			},
		},
	}
}

func toV2ListVolumesRequest(input ListVolumesInput) operations.V2GetVolumesWithBalancesRequest {
	request := operations.V2GetVolumesWithBalancesRequest{
		Ledger:        input.Ledger,
		PageSize:      pointer(input.PageSize),
		InsertionDate: pointer(input.UseInsertionDate),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	if input.StartTime != nil {
		request.StartTime = input.StartTime
	}
	if input.EndTime != nil {
		request.EndTime = input.EndTime
	}
	if input.GroupBy > 0 {
		request.GroupBy = pointer(input.GroupBy)
	}
	if input.Sort != "" {
		request.Sort = pointer(input.Sort)
	}

	queryParts := make([]map[string]any, 0, len(input.Metadata)+1)
	if input.Account != "" {
		queryParts = append(queryParts, map[string]any{
			"$match": map[string]any{"account": input.Account},
		})
	}
	for key, value := range input.Metadata {
		queryParts = append(queryParts, map[string]any{
			"$match": map[string]any{"metadata[" + key + "]": value},
		})
	}
	if len(queryParts) == 1 {
		request.Query = queryParts[0]
	}
	if len(queryParts) > 1 {
		request.Query = map[string]any{"$and": queryParts}
	}

	return request
}

func fromV2ListVolumes(cursor shared.V2VolumesWithBalanceCursorResponseCursor) ListVolumesOutput {
	volumes := make([]VolumeBalance, 0, len(cursor.Data))
	for _, volume := range cursor.Data {
		volumes = append(volumes, VolumeBalance{
			Account: volume.Account,
			Asset:   volume.Asset,
			Input:   bigIntString(volume.Input),
			Output:  bigIntString(volume.Output),
			Balance: bigIntString(volume.Balance),
		})
	}
	return ListVolumesOutput{
		Volumes:  volumes,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}
