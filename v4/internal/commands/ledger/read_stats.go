package ledger

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const FeatureReadStats capabilities.Feature = "readStats"

type ReadStatsInput struct {
	Ledger string
}

type ReadStatsOutput struct {
	APIVersion   capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Accounts     int64                   `json:"accounts" yaml:"accounts"`
	Transactions string                  `json:"transactions" yaml:"transactions"`
}

type ReadStatsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ReadStatsInput) (ReadStatsOutput, error)
}

type ReadStatsService struct {
	Handlers []ReadStatsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ReadStatsService) Run(ctx context.Context, input ReadStatsInput) (ReadStatsOutput, error) {
	if input.Ledger == "" {
		return ReadStatsOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ReadStatsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ReadStatsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ReadStatsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ReadStatsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKReadStatsHandlers(sdk *formance.Formance) []ReadStatsHandler {
	return []ReadStatsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ReadStatsInput) (ReadStatsOutput, error) {
				response, err := sdk.Ledger.V1.ReadStats(ctx, operations.ReadStatsRequest{Ledger: input.Ledger})
				if err != nil {
					return ReadStatsOutput{}, err
				}
				if response.StatsResponse == nil {
					return ReadStatsOutput{}, fmt.Errorf("ledger v1 read stats returned no data")
				}
				return fromV1Stats(response.StatsResponse.Data), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ReadStatsInput) (ReadStatsOutput, error) {
				response, err := sdk.Ledger.V2.ReadStats(ctx, operations.V2ReadStatsRequest{Ledger: input.Ledger})
				if err != nil {
					return ReadStatsOutput{}, err
				}
				if response.V2StatsResponse == nil {
					return ReadStatsOutput{}, fmt.Errorf("ledger v2 read stats returned no data")
				}
				return fromV2Stats(response.V2StatsResponse.Data), nil
			},
		},
	}
}

func fromV1Stats(stats shared.Stats) ReadStatsOutput {
	return ReadStatsOutput{
		Accounts:     stats.Accounts,
		Transactions: fmt.Sprintf("%d", stats.Transactions),
	}
}

func fromV2Stats(stats shared.V2Stats) ReadStatsOutput {
	return ReadStatsOutput{
		Accounts:     stats.Accounts,
		Transactions: bigIntString(stats.Transactions),
	}
}
