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

const (
	FeatureCreateLedger         capabilities.Feature = "createLedger"
	FeatureDeleteLedgerMetadata capabilities.Feature = "deleteLedgerMetadata"
	FeatureExportLogs           capabilities.Feature = "exportLogs"
	FeatureImportLogs           capabilities.Feature = "importLogs"
	FeatureListLedgers          capabilities.Feature = "listLedgers"
	FeatureUpdateLedgerMetadata capabilities.Feature = "updateLedgerMetadata"
)

type ListLedgersInput struct {
	PageSize       int64
	Cursor         string
	IncludeDeleted bool
	Sort           string
}

type CreateLedgerInput struct {
	Name     string
	Bucket   string
	Features map[string]string
	Metadata map[string]string
}

type CreateLedgerOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Name       string                  `json:"name" yaml:"name"`
	Created    bool                    `json:"created" yaml:"created"`
}

type ListLedgersOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Ledgers    []LedgerSummary         `json:"ledgers" yaml:"ledgers"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type LedgerSummary struct {
	Name      string            `json:"name" yaml:"name"`
	Bucket    string            `json:"bucket" yaml:"bucket"`
	AddedAt   time.Time         `json:"addedAt" yaml:"addedAt"`
	DeletedAt *time.Time        `json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListLedgersHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListLedgersInput) (ListLedgersOutput, error)
}

type ListLedgersService struct {
	Handlers []ListLedgersHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateLedgerHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateLedgerInput) (CreateLedgerOutput, error)
}

type CreateLedgerService struct {
	Handlers []CreateLedgerHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListLedgersService) Run(ctx context.Context, input ListLedgersInput) (ListLedgersOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListLedgersHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListLedgersOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListLedgersOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListLedgersOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreateLedgerService) Run(ctx context.Context, input CreateLedgerInput) (CreateLedgerOutput, error) {
	if input.Name == "" {
		return CreateLedgerOutput{}, fmt.Errorf("ledger name is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateLedgerHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateLedgerOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateLedgerOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateLedgerOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListLedgersHandlers(sdk *formance.Formance) []ListLedgersHandler {
	return []ListLedgersHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListLedgersInput) (ListLedgersOutput, error) {
				response, err := sdk.Ledger.V2.ListLedgers(ctx, toV2ListLedgersRequest(input))
				if err != nil {
					return ListLedgersOutput{}, err
				}
				if response.V2LedgerListResponse == nil {
					return ListLedgersOutput{}, fmt.Errorf("ledger v2 list ledgers returned no cursor")
				}
				return fromV2ListLedgers(response.V2LedgerListResponse.Cursor), nil
			},
		},
	}
}

func SDKCreateLedgerHandlers(sdk *formance.Formance) []CreateLedgerHandler {
	return []CreateLedgerHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input CreateLedgerInput) (CreateLedgerOutput, error) {
				request := operations.V2CreateLedgerRequest{
					Ledger: input.Name,
					V2CreateLedgerRequest: shared.V2CreateLedgerRequest{
						Features: input.Features,
						Metadata: input.Metadata,
					},
				}
				if input.Bucket != "" {
					request.V2CreateLedgerRequest.Bucket = pointer(input.Bucket)
				}
				response, err := sdk.Ledger.V2.CreateLedger(ctx, request)
				if err != nil {
					return CreateLedgerOutput{}, err
				}
				return CreateLedgerOutput{
					Name:    input.Name,
					Created: response.StatusCode == 204,
				}, nil
			},
		},
	}
}

func toV2ListLedgersRequest(input ListLedgersInput) operations.V2ListLedgersRequest {
	request := operations.V2ListLedgersRequest{
		PageSize:       pointer(input.PageSize),
		IncludeDeleted: pointer(input.IncludeDeleted),
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
	}
	if input.Sort != "" {
		request.Sort = pointer(input.Sort)
	}
	return request
}

func fromV2ListLedgers(cursor shared.V2LedgerListResponseCursor) ListLedgersOutput {
	ledgers := make([]LedgerSummary, 0, len(cursor.Data))
	for _, ledger := range cursor.Data {
		ledgers = append(ledgers, LedgerSummary{
			Name:      ledger.Name,
			Bucket:    ledger.Bucket,
			AddedAt:   ledger.AddedAt,
			DeletedAt: ledger.DeletedAt,
			Metadata:  ledger.Metadata,
		})
	}
	return ListLedgersOutput{
		Ledgers:  ledgers,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}
