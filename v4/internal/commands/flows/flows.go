package flows

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
	ProductOrchestration capabilities.Product = "orchestration"
	FeatureGetWorkflow   capabilities.Feature = "getWorkflow"
	FeatureListWorkflows capabilities.Feature = "listWorkflows"
)

type WorkflowSummary struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

type ListWorkflowsInput struct {
	PageSize int64
	Cursor   string
}

type ListWorkflowsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Workflows  []WorkflowSummary       `json:"workflows" yaml:"workflows"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetWorkflowInput struct {
	WorkflowID string
}

type GetWorkflowOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Workflow   WorkflowSummary         `json:"workflow" yaml:"workflow"`
}

type ListWorkflowsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListWorkflowsInput) (ListWorkflowsOutput, error)
}

type GetWorkflowHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetWorkflowInput) (GetWorkflowOutput, error)
}

type ListWorkflowsService struct {
	Handlers []ListWorkflowsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetWorkflowService struct {
	Handlers []GetWorkflowHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListWorkflowsService) Run(ctx context.Context, input ListWorkflowsInput) (ListWorkflowsOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListWorkflowsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListWorkflowsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListWorkflowsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListWorkflowsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetWorkflowService) Run(ctx context.Context, input GetWorkflowInput) (GetWorkflowOutput, error) {
	if input.WorkflowID == "" {
		return GetWorkflowOutput{}, fmt.Errorf("workflow id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetWorkflowHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetWorkflowOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetWorkflowOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetWorkflowOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKListWorkflowsHandlers(sdk *formance.Formance) []ListWorkflowsHandler {
	return []ListWorkflowsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, _ ListWorkflowsInput) (ListWorkflowsOutput, error) {
				response, err := sdk.Orchestration.V1.ListWorkflows(ctx)
				if err != nil {
					return ListWorkflowsOutput{}, err
				}
				if response.ListWorkflowsResponse == nil {
					return ListWorkflowsOutput{}, fmt.Errorf("orchestration v1 list workflows returned no data")
				}
				return fromV1Workflows(response.ListWorkflowsResponse.Data), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListWorkflowsInput) (ListWorkflowsOutput, error) {
				response, err := sdk.Orchestration.V2.ListWorkflows(ctx, operations.V2ListWorkflowsRequest{
					PageSize: optionalInt64(input.PageSize),
					Cursor:   optionalString(input.Cursor),
				})
				if err != nil {
					return ListWorkflowsOutput{}, err
				}
				if response.V2ListWorkflowsResponse == nil {
					return ListWorkflowsOutput{}, fmt.Errorf("orchestration v2 list workflows returned no cursor")
				}
				return fromV2WorkflowsCursor(response.V2ListWorkflowsResponse.Cursor), nil
			},
		},
	}
}

func SDKGetWorkflowHandlers(sdk *formance.Formance) []GetWorkflowHandler {
	return []GetWorkflowHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetWorkflowInput) (GetWorkflowOutput, error) {
				response, err := sdk.Orchestration.V1.GetWorkflow(ctx, operations.GetWorkflowRequest{FlowID: input.WorkflowID})
				if err != nil {
					return GetWorkflowOutput{}, err
				}
				if response.GetWorkflowResponse == nil {
					return GetWorkflowOutput{}, fmt.Errorf("orchestration v1 get workflow returned no data")
				}
				return GetWorkflowOutput{Workflow: fromV1Workflow(response.GetWorkflowResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetWorkflowInput) (GetWorkflowOutput, error) {
				response, err := sdk.Orchestration.V2.GetWorkflow(ctx, operations.V2GetWorkflowRequest{FlowID: input.WorkflowID})
				if err != nil {
					return GetWorkflowOutput{}, err
				}
				if response.V2GetWorkflowResponse == nil {
					return GetWorkflowOutput{}, fmt.Errorf("orchestration v2 get workflow returned no data")
				}
				return GetWorkflowOutput{Workflow: fromV2Workflow(response.V2GetWorkflowResponse.Data)}, nil
			},
		},
	}
}

func fromV1Workflows(workflows []shared.Workflow) ListWorkflowsOutput {
	ret := make([]WorkflowSummary, 0, len(workflows))
	for _, workflow := range workflows {
		ret = append(ret, fromV1Workflow(workflow))
	}
	return ListWorkflowsOutput{Workflows: ret, PageSize: int64(len(ret))}
}

func fromV2WorkflowsCursor(cursor shared.V2ListWorkflowsResponseCursor) ListWorkflowsOutput {
	workflows := make([]WorkflowSummary, 0, len(cursor.Data))
	for _, workflow := range cursor.Data {
		workflows = append(workflows, fromV2Workflow(workflow))
	}
	return ListWorkflowsOutput{
		Workflows: workflows,
		HasMore:   cursor.HasMore,
		PageSize:  cursor.PageSize,
		Next:      cursor.Next,
		Previous:  cursor.Previous,
	}
}

func fromV1Workflow(workflow shared.Workflow) WorkflowSummary {
	name := ""
	if workflow.Config.Name != nil {
		name = *workflow.Config.Name
	}
	return WorkflowSummary{
		ID:        workflow.ID,
		Name:      name,
		CreatedAt: workflow.CreatedAt,
		UpdatedAt: workflow.UpdatedAt,
	}
}

func fromV2Workflow(workflow shared.V2Workflow) WorkflowSummary {
	name := ""
	if workflow.Config.Name != nil {
		name = *workflow.Config.Name
	}
	return WorkflowSummary{
		ID:        workflow.ID,
		Name:      name,
		CreatedAt: workflow.CreatedAt,
		UpdatedAt: workflow.UpdatedAt,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
