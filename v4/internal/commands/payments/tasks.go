package payments

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const FeatureGetTask capabilities.Feature = "getTask"

type TaskSummary struct {
	ID              string    `json:"id" yaml:"id"`
	ConnectorID     string    `json:"connectorID,omitempty" yaml:"connectorID,omitempty"`
	CreatedObjectID string    `json:"createdObjectID,omitempty" yaml:"createdObjectID,omitempty"`
	Status          string    `json:"status" yaml:"status"`
	Error           string    `json:"error,omitempty" yaml:"error,omitempty"`
	CreatedAt       time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type GetTaskInput struct {
	TaskID string
}

type GetTaskOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Task       TaskSummary             `json:"task" yaml:"task"`
}

type GetTaskHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetTaskInput) (GetTaskOutput, error)
}

type GetTaskService struct {
	Handlers []GetTaskHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s GetTaskService) Run(ctx context.Context, input GetTaskInput) (GetTaskOutput, error) {
	if input.TaskID == "" {
		return GetTaskOutput{}, fmt.Errorf("task id is required")
	}
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]GetTaskHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetTaskOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetTaskOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetTaskOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKGetTaskHandlers(sdk *formance.Formance) []GetTaskHandler {
	return []GetTaskHandler{
		{
			APIVersion: "v3",
			Run: func(ctx context.Context, input GetTaskInput) (GetTaskOutput, error) {
				response, err := sdk.Payments.V3.GetTask(ctx, operations.V3GetTaskRequest{TaskID: input.TaskID})
				if err != nil {
					return GetTaskOutput{}, err
				}
				if response.V3GetTaskResponse == nil {
					return GetTaskOutput{}, fmt.Errorf("payments v3 get task returned no data")
				}
				return GetTaskOutput{Task: fromV3Task(response.V3GetTaskResponse.Data)}, nil
			},
		},
	}
}

func fromV3Task(task shared.V3Task) TaskSummary {
	return TaskSummary{
		ID:              task.ID,
		ConnectorID:     stringValue(task.ConnectorID),
		CreatedObjectID: stringValue(task.CreatedObjectID),
		Status:          string(task.Status),
		Error:           stringValue(task.Error),
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
}
