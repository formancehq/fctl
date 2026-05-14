package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestReadInfoServiceSelectsResolvedHandler(t *testing.T) {
	service := ReadInfoService{
		Handlers: []ReadInfoHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context) (ReadInfoOutput, error) {
					return ReadInfoOutput{Server: "ledger", Version: "1.0.0"}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1"})
			return "v1", nil
		},
	}

	output, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v1" || output.Server != "ledger" || output.Version != "1.0.0" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestFromV1Info(t *testing.T) {
	output := fromV1Info(shared.ConfigInfo{Server: "ledger", Version: "1.2.3"})
	if output.Server != "ledger" || output.Version != "1.2.3" {
		t.Fatalf("unexpected output: %#v", output)
	}
}
