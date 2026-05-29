package payments

import (
	"context"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListConnectorsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListConnectorsService{
		Handlers: []ListConnectorsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListConnectorsInput) (ListConnectorsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListConnectorsOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ListConnectorsInput) (ListConnectorsOutput, error) {
					if input.PageSize != 10 || input.Cursor != "cursor" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListConnectorsOutput{
						PageSize: input.PageSize,
						Connectors: []ConnectorSummary{
							{ID: "conn_1", Name: "Stripe", Provider: "stripe"},
						},
					}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListConnectorsInput{PageSize: 10, Cursor: "cursor"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 || output.Connectors[0].ID != "conn_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestUninstallConnectorServiceSelectsResolvedHandler(t *testing.T) {
	service := UninstallConnectorService{
		Handlers: []UninstallConnectorHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, UninstallConnectorInput) (UninstallConnectorOutput, error) {
					t.Fatal("v1 handler should not run")
					return UninstallConnectorOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input UninstallConnectorInput) (UninstallConnectorOutput, error) {
					if input.ConnectorID != "conn_1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return UninstallConnectorOutput{ConnectorID: input.ConnectorID, TaskID: "task_1"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), UninstallConnectorInput{ConnectorID: "conn_1"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.ConnectorID != "conn_1" || output.TaskID != "task_1" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestUninstallConnectorServiceRequiresConnectorID(t *testing.T) {
	service := UninstallConnectorService{
		Handlers: []UninstallConnectorHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), UninstallConnectorInput{}); err == nil {
		t.Fatal("expected connector id validation error")
	}
}

func TestUninstallConnectorV1HandlerRequiresProvider(t *testing.T) {
	handler := SDKUninstallConnectorHandlers(nil)[0]
	_, err := handler.Run(context.Background(), UninstallConnectorInput{ConnectorID: "conn_1"})
	if err == nil {
		t.Fatal("expected provider validation error")
	}
	if !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
