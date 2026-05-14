package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestExportLogsServiceSelectsResolvedHandler(t *testing.T) {
	service := ExportLogsService{
		Handlers: []ExportLogsHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ExportLogsInput) (ExportLogsOutput, error) {
					if input.Ledger != "default" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ExportLogsOutput{Ledger: input.Ledger, Data: []byte("entry\n")}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ExportLogsInput{Ledger: "default"})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Ledger != "default" || string(output.Data) != "entry\n" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestExportLogsServiceRequiresLedger(t *testing.T) {
	service := ExportLogsService{
		Handlers: []ExportLogsHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), ExportLogsInput{}); err == nil {
		t.Fatal("expected ledger validation error")
	}
}
