package payments

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestGetTaskServiceSelectsResolvedHandler(t *testing.T) {
	service := GetTaskService{
		Handlers: []GetTaskHandler{
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input GetTaskInput) (GetTaskOutput, error) {
					if input.TaskID != "task_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return GetTaskOutput{Task: TaskSummary{ID: input.TaskID}}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), GetTaskInput{TaskID: "task_1"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.Task.ID != "task_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetTaskServiceRequiresID(t *testing.T) {
	service := GetTaskService{
		Handlers: []GetTaskHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetTaskInput{}); err == nil {
		t.Fatal("expected task id validation error")
	}
}
