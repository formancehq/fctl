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
	ProductOrchestration  capabilities.Product = "orchestration"
	FeatureCreateWorkflow capabilities.Feature = "createWorkflow"
	FeatureDeleteWorkflow capabilities.Feature = "deleteWorkflow"
	FeatureGetWorkflow    capabilities.Feature = "getWorkflow"
	FeatureListWorkflows  capabilities.Feature = "listWorkflows"
	FeatureRunWorkflow    capabilities.Feature = "runWorkflow"
)

type WorkflowSummary struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

type InstanceSummary struct {
	ID           string     `json:"id" yaml:"id"`
	WorkflowID   string     `json:"workflowID" yaml:"workflowID"`
	CreatedAt    time.Time  `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	Terminated   bool       `json:"terminated" yaml:"terminated"`
	TerminatedAt *time.Time `json:"terminatedAt,omitempty" yaml:"terminatedAt,omitempty"`
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

type CreateWorkflowInput struct {
	Workflow shared.CreateWorkflowRequest
}

type CreateWorkflowOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Workflow   WorkflowSummary         `json:"workflow" yaml:"workflow"`
}

type DeleteWorkflowInput struct {
	WorkflowID string
}

type DeleteWorkflowOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	WorkflowID string                  `json:"workflowID" yaml:"workflowID"`
}

type RunWorkflowInput struct {
	WorkflowID string
	Vars       map[string]string
	Wait       *bool
}

type RunWorkflowOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Instance   InstanceSummary         `json:"instance" yaml:"instance"`
}

type ListWorkflowsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListWorkflowsInput) (ListWorkflowsOutput, error)
}

type GetWorkflowHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetWorkflowInput) (GetWorkflowOutput, error)
}

type CreateWorkflowHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateWorkflowInput) (CreateWorkflowOutput, error)
}

type DeleteWorkflowHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteWorkflowInput) (DeleteWorkflowOutput, error)
}

type RunWorkflowHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, RunWorkflowInput) (RunWorkflowOutput, error)
}

type ListWorkflowsService struct {
	Handlers []ListWorkflowsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetWorkflowService struct {
	Handlers []GetWorkflowHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateWorkflowService struct {
	Handlers []CreateWorkflowHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteWorkflowService struct {
	Handlers []DeleteWorkflowHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type RunWorkflowService struct {
	Handlers []RunWorkflowHandler
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

func (s CreateWorkflowService) Run(ctx context.Context, input CreateWorkflowInput) (CreateWorkflowOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateWorkflowHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateWorkflowOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateWorkflowOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateWorkflowOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s DeleteWorkflowService) Run(ctx context.Context, input DeleteWorkflowInput) (DeleteWorkflowOutput, error) {
	if input.WorkflowID == "" {
		return DeleteWorkflowOutput{}, fmt.Errorf("workflow id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeleteWorkflowHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeleteWorkflowOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeleteWorkflowOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteWorkflowOutput{}, err
	}
	output.APIVersion = selected
	if output.WorkflowID == "" {
		output.WorkflowID = input.WorkflowID
	}
	return output, nil
}

func (s RunWorkflowService) Run(ctx context.Context, input RunWorkflowInput) (RunWorkflowOutput, error) {
	if input.WorkflowID == "" {
		return RunWorkflowOutput{}, fmt.Errorf("workflow id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]RunWorkflowHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return RunWorkflowOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return RunWorkflowOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return RunWorkflowOutput{}, err
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

func SDKCreateWorkflowHandlers(sdk *formance.Formance) []CreateWorkflowHandler {
	return []CreateWorkflowHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateWorkflowInput) (CreateWorkflowOutput, error) {
				response, err := sdk.Orchestration.V1.CreateWorkflow(ctx, &input.Workflow)
				if err != nil {
					return CreateWorkflowOutput{}, err
				}
				if response.CreateWorkflowResponse == nil {
					return CreateWorkflowOutput{}, fmt.Errorf("orchestration v1 create workflow returned no data")
				}
				return CreateWorkflowOutput{Workflow: fromV1Workflow(response.CreateWorkflowResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input CreateWorkflowInput) (CreateWorkflowOutput, error) {
				request := shared.V2CreateWorkflowRequest{
					Name:   input.Workflow.Name,
					Stages: input.Workflow.Stages,
				}
				response, err := sdk.Orchestration.V2.CreateWorkflow(ctx, &request)
				if err != nil {
					return CreateWorkflowOutput{}, err
				}
				if response.V2CreateWorkflowResponse == nil {
					return CreateWorkflowOutput{}, fmt.Errorf("orchestration v2 create workflow returned no data")
				}
				return CreateWorkflowOutput{Workflow: fromV2Workflow(response.V2CreateWorkflowResponse.Data)}, nil
			},
		},
	}
}

func SDKDeleteWorkflowHandlers(sdk *formance.Formance) []DeleteWorkflowHandler {
	return []DeleteWorkflowHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeleteWorkflowInput) (DeleteWorkflowOutput, error) {
				_, err := sdk.Orchestration.V1.DeleteWorkflow(ctx, operations.DeleteWorkflowRequest{FlowID: input.WorkflowID})
				if err != nil {
					return DeleteWorkflowOutput{}, err
				}
				return DeleteWorkflowOutput{WorkflowID: input.WorkflowID}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input DeleteWorkflowInput) (DeleteWorkflowOutput, error) {
				_, err := sdk.Orchestration.V2.DeleteWorkflow(ctx, operations.V2DeleteWorkflowRequest{FlowID: input.WorkflowID})
				if err != nil {
					return DeleteWorkflowOutput{}, err
				}
				return DeleteWorkflowOutput{WorkflowID: input.WorkflowID}, nil
			},
		},
	}
}

func SDKRunWorkflowHandlers(sdk *formance.Formance) []RunWorkflowHandler {
	return []RunWorkflowHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input RunWorkflowInput) (RunWorkflowOutput, error) {
				response, err := sdk.Orchestration.V1.RunWorkflow(ctx, operations.RunWorkflowRequest{
					WorkflowID:  input.WorkflowID,
					RequestBody: input.Vars,
					Wait:        input.Wait,
				})
				if err != nil {
					return RunWorkflowOutput{}, err
				}
				if response.RunWorkflowResponse == nil {
					return RunWorkflowOutput{}, fmt.Errorf("orchestration v1 run workflow returned no data")
				}
				return RunWorkflowOutput{Instance: fromV1Instance(response.RunWorkflowResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input RunWorkflowInput) (RunWorkflowOutput, error) {
				response, err := sdk.Orchestration.V2.RunWorkflow(ctx, operations.V2RunWorkflowRequest{
					WorkflowID:  input.WorkflowID,
					RequestBody: input.Vars,
					Wait:        input.Wait,
				})
				if err != nil {
					return RunWorkflowOutput{}, err
				}
				if response.V2RunWorkflowResponse == nil {
					return RunWorkflowOutput{}, fmt.Errorf("orchestration v2 run workflow returned no data")
				}
				return RunWorkflowOutput{Instance: fromV2Instance(response.V2RunWorkflowResponse.Data)}, nil
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

func fromV1Instance(instance shared.WorkflowInstance) InstanceSummary {
	return InstanceSummary{
		ID:           instance.ID,
		WorkflowID:   instance.WorkflowID,
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
		Terminated:   instance.Terminated,
		TerminatedAt: instance.TerminatedAt,
	}
}

func fromV2Instance(instance shared.V2WorkflowInstance) InstanceSummary {
	return InstanceSummary{
		ID:           instance.ID,
		WorkflowID:   instance.WorkflowID,
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
		Terminated:   instance.Terminated,
		TerminatedAt: instance.TerminatedAt,
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
