package flows

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListWorkflowsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListWorkflowsService{
		Handlers: []ListWorkflowsHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListWorkflowsInput) (ListWorkflowsOutput, error) {
					if input.PageSize != 10 || input.Cursor != "cursor" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListWorkflowsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListWorkflowsInput{PageSize: 10, Cursor: "cursor"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetWorkflowServiceRequiresWorkflowID(t *testing.T) {
	service := GetWorkflowService{
		Handlers: []GetWorkflowHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetWorkflowInput{}); err == nil {
		t.Fatal("expected workflow id validation error")
	}
}

func TestDeleteWorkflowServiceRequiresWorkflowID(t *testing.T) {
	service := DeleteWorkflowService{
		Handlers: []DeleteWorkflowHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), DeleteWorkflowInput{}); err == nil {
		t.Fatal("expected workflow id validation error")
	}
}

func TestRunWorkflowServiceSelectsResolvedHandler(t *testing.T) {
	service := RunWorkflowService{
		Handlers: []RunWorkflowHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input RunWorkflowInput) (RunWorkflowOutput, error) {
					if input.WorkflowID != "workflow_1" || input.Vars["env"] != "dev" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return RunWorkflowOutput{Instance: InstanceSummary{ID: "instance_1"}}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), RunWorkflowInput{WorkflowID: "workflow_1", Vars: map[string]string{"env": "dev"}})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Instance.ID != "instance_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestListInstancesServiceSelectsResolvedHandler(t *testing.T) {
	service := ListInstancesService{
		Handlers: []ListInstancesHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListInstancesInput) (ListInstancesOutput, error) {
					if input.PageSize != 10 || input.WorkflowID != "workflow_1" || input.Running == nil || !*input.Running {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListInstancesOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	running := true
	output, err := service.Run(context.Background(), ListInstancesInput{PageSize: 10, WorkflowID: "workflow_1", Running: &running})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetInstanceServiceRequiresInstanceID(t *testing.T) {
	service := GetInstanceService{
		Handlers: []GetInstanceHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetInstanceInput{}); err == nil {
		t.Fatal("expected instance id validation error")
	}
}

func assertAPIVersions(t *testing.T, got []capabilities.APIVersion, want []capabilities.APIVersion) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected versions %v, got %v", want, got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("expected versions %v, got %v", want, got)
		}
	}
}
