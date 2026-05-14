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
	FeatureCancelEvent    capabilities.Feature = "cancelEvent"
	FeatureCreateTrigger  capabilities.Feature = "createTrigger"
	FeatureCreateWorkflow capabilities.Feature = "createWorkflow"
	FeatureDeleteTrigger  capabilities.Feature = "deleteTrigger"
	FeatureDeleteWorkflow capabilities.Feature = "deleteWorkflow"
	FeatureGetInstance    capabilities.Feature = "getInstance"
	FeatureGetWorkflow    capabilities.Feature = "getWorkflow"
	FeatureListInstances  capabilities.Feature = "listInstances"
	FeatureListTriggers   capabilities.Feature = "listTriggers"
	FeatureListWorkflows  capabilities.Feature = "listWorkflows"
	FeatureReadTrigger    capabilities.Feature = "readTrigger"
	FeatureRunWorkflow    capabilities.Feature = "runWorkflow"
	FeatureSendEvent      capabilities.Feature = "sendEvent"
	FeatureTestTrigger    capabilities.Feature = "testTrigger"
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

type TriggerSummary struct {
	ID         string    `json:"id" yaml:"id"`
	Name       string    `json:"name,omitempty" yaml:"name,omitempty"`
	Event      string    `json:"event" yaml:"event"`
	WorkflowID string    `json:"workflowID" yaml:"workflowID"`
	CreatedAt  time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	Version    string    `json:"version,omitempty" yaml:"version,omitempty"`
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

type ListInstancesInput struct {
	PageSize   int64
	Cursor     string
	WorkflowID string
	Running    *bool
}

type ListInstancesOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Instances  []InstanceSummary       `json:"instances" yaml:"instances"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetInstanceInput struct {
	InstanceID string
}

type GetInstanceOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Instance   InstanceSummary         `json:"instance" yaml:"instance"`
}

type InstanceActionInput struct {
	InstanceID string
	Event      string
}

type InstanceActionOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	InstanceID string                  `json:"instanceID" yaml:"instanceID"`
	Event      string                  `json:"event,omitempty" yaml:"event,omitempty"`
}

type ListTriggersInput struct {
	PageSize int64
	Cursor   string
	Name     string
}

type ListTriggersOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Triggers   []TriggerSummary        `json:"triggers" yaml:"triggers"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetTriggerInput struct {
	TriggerID string
}

type GetTriggerOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Trigger    TriggerSummary          `json:"trigger" yaml:"trigger"`
}

type CreateTriggerInput struct {
	Event      string
	WorkflowID string
	Name       string
	Filter     string
	Version    string
	Vars       map[string]any
}

type CreateTriggerOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Trigger    TriggerSummary          `json:"trigger" yaml:"trigger"`
}

type DeleteTriggerInput struct {
	TriggerID string
}

type DeleteTriggerOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	TriggerID  string                  `json:"triggerID" yaml:"triggerID"`
}

type TestTriggerInput struct {
	TriggerID string
	Event     map[string]any
}

type TestTriggerOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	TriggerID  string                  `json:"triggerID" yaml:"triggerID"`
	Matched    *bool                   `json:"matched,omitempty" yaml:"matched,omitempty"`
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

type ListInstancesHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListInstancesInput) (ListInstancesOutput, error)
}

type GetInstanceHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetInstanceInput) (GetInstanceOutput, error)
}

type InstanceActionHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, InstanceActionInput) (InstanceActionOutput, error)
}

type ListTriggersHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListTriggersInput) (ListTriggersOutput, error)
}

type GetTriggerHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetTriggerInput) (GetTriggerOutput, error)
}

type CreateTriggerHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateTriggerInput) (CreateTriggerOutput, error)
}

type DeleteTriggerHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteTriggerInput) (DeleteTriggerOutput, error)
}

type TestTriggerHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, TestTriggerInput) (TestTriggerOutput, error)
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

type ListInstancesService struct {
	Handlers []ListInstancesHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetInstanceService struct {
	Handlers []GetInstanceHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type SendEventService struct {
	Handlers []InstanceActionHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type StopInstanceService struct {
	Handlers []InstanceActionHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListTriggersService struct {
	Handlers []ListTriggersHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetTriggerService struct {
	Handlers []GetTriggerHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateTriggerService struct {
	Handlers []CreateTriggerHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteTriggerService struct {
	Handlers []DeleteTriggerHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type TestTriggerService struct {
	Handlers []TestTriggerHandler
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

func (s ListInstancesService) Run(ctx context.Context, input ListInstancesInput) (ListInstancesOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListInstancesHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListInstancesOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListInstancesOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListInstancesOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetInstanceService) Run(ctx context.Context, input GetInstanceInput) (GetInstanceOutput, error) {
	if input.InstanceID == "" {
		return GetInstanceOutput{}, fmt.Errorf("instance id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetInstanceHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetInstanceOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetInstanceOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetInstanceOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s SendEventService) Run(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
	if input.Event == "" {
		return InstanceActionOutput{}, fmt.Errorf("event is required")
	}
	return runInstanceActionService(ctx, input, s.Handlers, s.Resolve)
}

func (s StopInstanceService) Run(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
	return runInstanceActionService(ctx, input, s.Handlers, s.Resolve)
}

func runInstanceActionService(
	ctx context.Context,
	input InstanceActionInput,
	handlers []InstanceActionHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (InstanceActionOutput, error) {
	if input.InstanceID == "" {
		return InstanceActionOutput{}, fmt.Errorf("instance id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(handlers))
	handlersByVersion := map[capabilities.APIVersion]InstanceActionHandler{}
	for _, handler := range handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlersByVersion[handler.APIVersion] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return InstanceActionOutput{}, err
	}
	handler, ok := handlersByVersion[selected]
	if !ok {
		return InstanceActionOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return InstanceActionOutput{}, err
	}
	output.APIVersion = selected
	if output.InstanceID == "" {
		output.InstanceID = input.InstanceID
	}
	if output.Event == "" {
		output.Event = input.Event
	}
	return output, nil
}

func (s ListTriggersService) Run(ctx context.Context, input ListTriggersInput) (ListTriggersOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ListTriggersHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListTriggersOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListTriggersOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListTriggersOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetTriggerService) Run(ctx context.Context, input GetTriggerInput) (GetTriggerOutput, error) {
	if input.TriggerID == "" {
		return GetTriggerOutput{}, fmt.Errorf("trigger id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetTriggerHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetTriggerOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetTriggerOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetTriggerOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreateTriggerService) Run(ctx context.Context, input CreateTriggerInput) (CreateTriggerOutput, error) {
	if input.Event == "" {
		return CreateTriggerOutput{}, fmt.Errorf("event is required")
	}
	if input.WorkflowID == "" {
		return CreateTriggerOutput{}, fmt.Errorf("workflow id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]CreateTriggerHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return CreateTriggerOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return CreateTriggerOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateTriggerOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s DeleteTriggerService) Run(ctx context.Context, input DeleteTriggerInput) (DeleteTriggerOutput, error) {
	if input.TriggerID == "" {
		return DeleteTriggerOutput{}, fmt.Errorf("trigger id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeleteTriggerHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeleteTriggerOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeleteTriggerOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteTriggerOutput{}, err
	}
	output.APIVersion = selected
	if output.TriggerID == "" {
		output.TriggerID = input.TriggerID
	}
	return output, nil
}

func (s TestTriggerService) Run(ctx context.Context, input TestTriggerInput) (TestTriggerOutput, error) {
	if input.TriggerID == "" {
		return TestTriggerOutput{}, fmt.Errorf("trigger id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]TestTriggerHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return TestTriggerOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return TestTriggerOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return TestTriggerOutput{}, err
	}
	output.APIVersion = selected
	if output.TriggerID == "" {
		output.TriggerID = input.TriggerID
	}
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

func SDKListInstancesHandlers(sdk *formance.Formance) []ListInstancesHandler {
	return []ListInstancesHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListInstancesInput) (ListInstancesOutput, error) {
				response, err := sdk.Orchestration.V1.ListInstances(ctx, operations.ListInstancesRequest{
					WorkflowID: optionalString(input.WorkflowID),
					Running:    input.Running,
				})
				if err != nil {
					return ListInstancesOutput{}, err
				}
				if response.ListRunsResponse == nil {
					return ListInstancesOutput{}, fmt.Errorf("orchestration v1 list instances returned no data")
				}
				return fromV1Instances(response.ListRunsResponse.Data), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListInstancesInput) (ListInstancesOutput, error) {
				response, err := sdk.Orchestration.V2.ListInstances(ctx, operations.V2ListInstancesRequest{
					PageSize:   optionalInt64(input.PageSize),
					Cursor:     optionalString(input.Cursor),
					WorkflowID: optionalString(input.WorkflowID),
					Running:    input.Running,
				})
				if err != nil {
					return ListInstancesOutput{}, err
				}
				if response.V2ListRunsResponse == nil {
					return ListInstancesOutput{}, fmt.Errorf("orchestration v2 list instances returned no cursor")
				}
				return fromV2InstancesCursor(response.V2ListRunsResponse.Cursor), nil
			},
		},
	}
}

func SDKGetInstanceHandlers(sdk *formance.Formance) []GetInstanceHandler {
	return []GetInstanceHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetInstanceInput) (GetInstanceOutput, error) {
				response, err := sdk.Orchestration.V1.GetInstance(ctx, operations.GetInstanceRequest{InstanceID: input.InstanceID})
				if err != nil {
					return GetInstanceOutput{}, err
				}
				if response.GetWorkflowInstanceResponse == nil {
					return GetInstanceOutput{}, fmt.Errorf("orchestration v1 get instance returned no data")
				}
				return GetInstanceOutput{Instance: fromV1Instance(response.GetWorkflowInstanceResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetInstanceInput) (GetInstanceOutput, error) {
				response, err := sdk.Orchestration.V2.GetInstance(ctx, operations.V2GetInstanceRequest{InstanceID: input.InstanceID})
				if err != nil {
					return GetInstanceOutput{}, err
				}
				if response.V2GetWorkflowInstanceResponse == nil {
					return GetInstanceOutput{}, fmt.Errorf("orchestration v2 get instance returned no data")
				}
				return GetInstanceOutput{Instance: fromV2Instance(response.V2GetWorkflowInstanceResponse.Data)}, nil
			},
		},
	}
}

func SDKSendEventHandlers(sdk *formance.Formance) []InstanceActionHandler {
	return []InstanceActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
				_, err := sdk.Orchestration.V1.SendEvent(ctx, operations.SendEventRequest{
					InstanceID: input.InstanceID,
					RequestBody: &operations.SendEventRequestBody{
						Name: input.Event,
					},
				})
				if err != nil {
					return InstanceActionOutput{}, err
				}
				return InstanceActionOutput{InstanceID: input.InstanceID, Event: input.Event}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
				_, err := sdk.Orchestration.V2.SendEvent(ctx, operations.V2SendEventRequest{
					InstanceID: input.InstanceID,
					RequestBody: &operations.V2SendEventRequestBody{
						Name: input.Event,
					},
				})
				if err != nil {
					return InstanceActionOutput{}, err
				}
				return InstanceActionOutput{InstanceID: input.InstanceID, Event: input.Event}, nil
			},
		},
	}
}

func SDKStopInstanceHandlers(sdk *formance.Formance) []InstanceActionHandler {
	return []InstanceActionHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
				_, err := sdk.Orchestration.V1.CancelEvent(ctx, operations.CancelEventRequest{InstanceID: input.InstanceID})
				if err != nil {
					return InstanceActionOutput{}, err
				}
				return InstanceActionOutput{InstanceID: input.InstanceID}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input InstanceActionInput) (InstanceActionOutput, error) {
				_, err := sdk.Orchestration.V2.CancelEvent(ctx, operations.V2CancelEventRequest{InstanceID: input.InstanceID})
				if err != nil {
					return InstanceActionOutput{}, err
				}
				return InstanceActionOutput{InstanceID: input.InstanceID}, nil
			},
		},
	}
}

func SDKListTriggersHandlers(sdk *formance.Formance) []ListTriggersHandler {
	return []ListTriggersHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListTriggersInput) (ListTriggersOutput, error) {
				response, err := sdk.Orchestration.V1.ListTriggers(ctx, operations.ListTriggersRequest{Name: optionalString(input.Name)})
				if err != nil {
					return ListTriggersOutput{}, err
				}
				if response.ListTriggersResponse == nil {
					return ListTriggersOutput{}, fmt.Errorf("orchestration v1 list triggers returned no data")
				}
				return fromV1Triggers(response.ListTriggersResponse.Data), nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListTriggersInput) (ListTriggersOutput, error) {
				response, err := sdk.Orchestration.V2.ListTriggers(ctx, operations.V2ListTriggersRequest{
					PageSize: optionalInt64(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Name:     optionalString(input.Name),
				})
				if err != nil {
					return ListTriggersOutput{}, err
				}
				if response.V2ListTriggersResponse == nil {
					return ListTriggersOutput{}, fmt.Errorf("orchestration v2 list triggers returned no cursor")
				}
				return fromV2TriggersCursor(response.V2ListTriggersResponse.Cursor), nil
			},
		},
	}
}

func SDKGetTriggerHandlers(sdk *formance.Formance) []GetTriggerHandler {
	return []GetTriggerHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetTriggerInput) (GetTriggerOutput, error) {
				response, err := sdk.Orchestration.V1.ReadTrigger(ctx, operations.ReadTriggerRequest{TriggerID: input.TriggerID})
				if err != nil {
					return GetTriggerOutput{}, err
				}
				if response.ReadTriggerResponse == nil {
					return GetTriggerOutput{}, fmt.Errorf("orchestration v1 read trigger returned no data")
				}
				return GetTriggerOutput{Trigger: fromV1Trigger(response.ReadTriggerResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetTriggerInput) (GetTriggerOutput, error) {
				response, err := sdk.Orchestration.V2.ReadTrigger(ctx, operations.V2ReadTriggerRequest{TriggerID: input.TriggerID})
				if err != nil {
					return GetTriggerOutput{}, err
				}
				if response.V2ReadTriggerResponse == nil {
					return GetTriggerOutput{}, fmt.Errorf("orchestration v2 read trigger returned no data")
				}
				return GetTriggerOutput{Trigger: fromV2Trigger(response.V2ReadTriggerResponse.Data)}, nil
			},
		},
	}
}

func SDKCreateTriggerHandlers(sdk *formance.Formance) []CreateTriggerHandler {
	return []CreateTriggerHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateTriggerInput) (CreateTriggerOutput, error) {
				response, err := sdk.Orchestration.V1.CreateTrigger(ctx, &shared.TriggerData{
					Event:      input.Event,
					WorkflowID: input.WorkflowID,
					Name:       optionalString(input.Name),
					Filter:     optionalString(input.Filter),
					Version:    optionalString(input.Version),
					Vars:       input.Vars,
				})
				if err != nil {
					return CreateTriggerOutput{}, err
				}
				if response.CreateTriggerResponse == nil {
					return CreateTriggerOutput{}, fmt.Errorf("orchestration v1 create trigger returned no data")
				}
				return CreateTriggerOutput{Trigger: fromV1Trigger(response.CreateTriggerResponse.Data)}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input CreateTriggerInput) (CreateTriggerOutput, error) {
				response, err := sdk.Orchestration.V2.CreateTrigger(ctx, &shared.V2TriggerData{
					Event:      input.Event,
					WorkflowID: input.WorkflowID,
					Name:       optionalString(input.Name),
					Filter:     optionalString(input.Filter),
					Version:    optionalString(input.Version),
					Vars:       input.Vars,
				})
				if err != nil {
					return CreateTriggerOutput{}, err
				}
				if response.V2CreateTriggerResponse == nil {
					return CreateTriggerOutput{}, fmt.Errorf("orchestration v2 create trigger returned no data")
				}
				return CreateTriggerOutput{Trigger: fromV2Trigger(response.V2CreateTriggerResponse.Data)}, nil
			},
		},
	}
}

func SDKDeleteTriggerHandlers(sdk *formance.Formance) []DeleteTriggerHandler {
	return []DeleteTriggerHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeleteTriggerInput) (DeleteTriggerOutput, error) {
				_, err := sdk.Orchestration.V1.DeleteTrigger(ctx, operations.DeleteTriggerRequest{TriggerID: input.TriggerID})
				if err != nil {
					return DeleteTriggerOutput{}, err
				}
				return DeleteTriggerOutput{TriggerID: input.TriggerID}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input DeleteTriggerInput) (DeleteTriggerOutput, error) {
				_, err := sdk.Orchestration.V2.DeleteTrigger(ctx, operations.V2DeleteTriggerRequest{TriggerID: input.TriggerID})
				if err != nil {
					return DeleteTriggerOutput{}, err
				}
				return DeleteTriggerOutput{TriggerID: input.TriggerID}, nil
			},
		},
	}
}

func SDKTestTriggerHandlers(sdk *formance.Formance) []TestTriggerHandler {
	return []TestTriggerHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input TestTriggerInput) (TestTriggerOutput, error) {
				response, err := sdk.Orchestration.V2.TestTrigger(ctx, operations.TestTriggerRequest{
					TriggerID:   input.TriggerID,
					RequestBody: input.Event,
				})
				if err != nil {
					return TestTriggerOutput{}, err
				}
				output := TestTriggerOutput{TriggerID: input.TriggerID}
				if response.V2TestTriggerResponse != nil && response.V2TestTriggerResponse.Data.Filter != nil {
					output.Matched = response.V2TestTriggerResponse.Data.Filter.Match
				}
				return output, nil
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

func fromV1Triggers(triggers []shared.Trigger) ListTriggersOutput {
	ret := make([]TriggerSummary, 0, len(triggers))
	for _, trigger := range triggers {
		ret = append(ret, fromV1Trigger(trigger))
	}
	return ListTriggersOutput{Triggers: ret, PageSize: int64(len(ret))}
}

func fromV2TriggersCursor(cursor shared.V2ListTriggersResponseCursor) ListTriggersOutput {
	triggers := make([]TriggerSummary, 0, len(cursor.Data))
	for _, trigger := range cursor.Data {
		triggers = append(triggers, fromV2Trigger(trigger))
	}
	return ListTriggersOutput{
		Triggers: triggers,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV1Trigger(trigger shared.Trigger) TriggerSummary {
	name := ""
	if trigger.Name != nil {
		name = *trigger.Name
	}
	version := ""
	if trigger.Version != nil {
		version = *trigger.Version
	}
	return TriggerSummary{
		ID:         trigger.ID,
		Name:       name,
		Event:      trigger.Event,
		WorkflowID: trigger.WorkflowID,
		CreatedAt:  trigger.CreatedAt,
		Version:    version,
	}
}

func fromV2Trigger(trigger shared.V2Trigger) TriggerSummary {
	name := ""
	if trigger.Name != nil {
		name = *trigger.Name
	}
	version := ""
	if trigger.Version != nil {
		version = *trigger.Version
	}
	return TriggerSummary{
		ID:         trigger.ID,
		Name:       name,
		Event:      trigger.Event,
		WorkflowID: trigger.WorkflowID,
		CreatedAt:  trigger.CreatedAt,
		Version:    version,
	}
}

func fromV1Instances(instances []shared.WorkflowInstance) ListInstancesOutput {
	ret := make([]InstanceSummary, 0, len(instances))
	for _, instance := range instances {
		ret = append(ret, fromV1Instance(instance))
	}
	return ListInstancesOutput{Instances: ret, PageSize: int64(len(ret))}
}

func fromV2InstancesCursor(cursor shared.V2ListRunsResponseCursor) ListInstancesOutput {
	instances := make([]InstanceSummary, 0, len(cursor.Data))
	for _, instance := range cursor.Data {
		instances = append(instances, fromV2Instance(instance))
	}
	return ListInstancesOutput{
		Instances: instances,
		HasMore:   cursor.HasMore,
		PageSize:  cursor.PageSize,
		Next:      cursor.Next,
		Previous:  cursor.Previous,
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
