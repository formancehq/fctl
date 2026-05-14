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

const FeatureRunQuery capabilities.Feature = "runQuery"

type RunAccountQueryInput struct {
	Ledger        string
	QueryID       string
	SchemaVersion string
	PageSize      int64
	Cursor        string
	Expand        string
	Pit           *time.Time
	Reverse       bool
	Sort          string
	Vars          map[string]string
}

type RunAccountQueryOutput struct {
	APIVersion    capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	QueryID       string                  `json:"queryId" yaml:"queryId"`
	SchemaVersion string                  `json:"schemaVersion" yaml:"schemaVersion"`
	Accounts      []AccountSummary        `json:"accounts" yaml:"accounts"`
	HasMore       bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize      int64                   `json:"pageSize" yaml:"pageSize"`
	Next          *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous      *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type RunAccountQueryHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, RunAccountQueryInput) (RunAccountQueryOutput, error)
}

type RunAccountQueryService struct {
	Handlers []RunAccountQueryHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s RunAccountQueryService) Run(ctx context.Context, input RunAccountQueryInput) (RunAccountQueryOutput, error) {
	if input.Ledger == "" {
		return RunAccountQueryOutput{}, fmt.Errorf("ledger is required")
	}
	if input.QueryID == "" {
		return RunAccountQueryOutput{}, fmt.Errorf("query id is required")
	}
	if input.SchemaVersion == "" {
		return RunAccountQueryOutput{}, fmt.Errorf("schema version is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]RunAccountQueryHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return RunAccountQueryOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return RunAccountQueryOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return RunAccountQueryOutput{}, err
	}
	output.APIVersion = selected
	output.QueryID = input.QueryID
	output.SchemaVersion = input.SchemaVersion
	return output, nil
}

func SDKRunAccountQueryHandlers(sdk *formance.Formance) []RunAccountQueryHandler {
	return []RunAccountQueryHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input RunAccountQueryInput) (RunAccountQueryOutput, error) {
				response, err := sdk.Ledger.V2.RunQuery(ctx, toV2RunAccountQueryRequest(input))
				if err != nil {
					return RunAccountQueryOutput{}, err
				}
				if response.OneOf == nil || response.OneOf.V2AccountsCursorResponse == nil {
					return RunAccountQueryOutput{}, fmt.Errorf("ledger v2 run query returned no accounts cursor")
				}
				return fromV2RunAccountQuery(response.OneOf.V2AccountsCursorResponse.Cursor), nil
			},
		},
	}
}

func toV2RunAccountQueryRequest(input RunAccountQueryInput) operations.V2RunQueryRequest {
	params := shared.QueryTemplateAccountParams{
		Resource: shared.V2QueryParamsResourceAccounts.ToPointer(),
	}
	request := operations.V2RunQueryRequest{
		ID:            input.QueryID,
		Ledger:        input.Ledger,
		SchemaVersion: input.SchemaVersion,
		RequestBody: operations.V2RunQueryRequestBody{
			Params: pointer(shared.CreateV2QueryParamsQueryTemplateAccountParams(params)),
			Vars:   input.Vars,
		},
	}
	if input.PageSize > 0 {
		request.PageSize = pointer(input.PageSize)
		params.PageSize = pointer(input.PageSize)
	}
	if input.Cursor != "" {
		request.Cursor = pointer(input.Cursor)
		request.RequestBody.Cursor = pointer(input.Cursor)
		params.Cursor = pointer(input.Cursor)
	}
	if input.Expand != "" {
		request.Expand = pointer(input.Expand)
		params.Expand = pointer(input.Expand)
	}
	if input.Pit != nil {
		request.Pit = input.Pit
		params.Pit = input.Pit
	}
	if input.Reverse {
		request.Reverse = pointer(input.Reverse)
	}
	if input.Sort != "" {
		request.Sort = pointer(input.Sort)
		params.Sort = pointer(input.Sort)
	}
	request.RequestBody.Params = pointer(shared.CreateV2QueryParamsQueryTemplateAccountParams(params))
	return request
}

func fromV2RunAccountQuery(cursor shared.V2AccountsCursorResponseCursor) RunAccountQueryOutput {
	accounts := make([]AccountSummary, 0, len(cursor.Data))
	for _, account := range cursor.Data {
		accounts = append(accounts, AccountSummary{
			Address:  account.Address,
			Metadata: stringMapToAny(account.Metadata),
		})
	}
	return RunAccountQueryOutput{
		Accounts: accounts,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}
