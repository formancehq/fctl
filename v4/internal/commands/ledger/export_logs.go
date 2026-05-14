package ledger

import (
	"context"
	"fmt"
	"os"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type ExportLogsInput struct {
	Ledger string
}

type ExportLogsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Ledger     string                  `json:"ledger" yaml:"ledger"`
	Data       []byte                  `json:"-" yaml:"-"`
}

type ExportLogsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ExportLogsInput) (ExportLogsOutput, error)
}

type ExportLogsService struct {
	Handlers []ExportLogsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ExportLogsService) Run(ctx context.Context, input ExportLogsInput) (ExportLogsOutput, error) {
	if input.Ledger == "" {
		return ExportLogsOutput{}, fmt.Errorf("ledger is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ExportLogsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ExportLogsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ExportLogsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ExportLogsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKExportLogsHandlers(sdk *formance.Formance) []ExportLogsHandler {
	return []ExportLogsHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ExportLogsInput) (ExportLogsOutput, error) {
				file, err := os.CreateTemp("", "fctl-ledger-export-*.jsonl")
				if err != nil {
					return ExportLogsOutput{}, err
				}
				path := file.Name()
				if err := file.Close(); err != nil {
					return ExportLogsOutput{}, err
				}
				defer os.Remove(path)

				ctx = context.WithValue(ctx, "path", path)
				response, err := sdk.Ledger.V2.ExportLogs(ctx, operations.V2ExportLogsRequest{
					Ledger: input.Ledger,
				})
				if err != nil {
					return ExportLogsOutput{}, err
				}
				if response.RawResponse == nil {
					return ExportLogsOutput{}, fmt.Errorf("ledger v2 export logs returned no response")
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return ExportLogsOutput{}, err
				}
				return ExportLogsOutput{Ledger: input.Ledger, Data: data}, nil
			},
		},
	}
}
