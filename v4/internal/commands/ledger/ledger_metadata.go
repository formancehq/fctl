package ledger

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type UpdateLedgerMetadataInput struct {
	Ledger   string
	Metadata map[string]string
}

type UpdateLedgerMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Updated    bool                    `json:"updated" yaml:"updated"`
}

type DeleteLedgerMetadataInput struct {
	Ledger string
	Key    string
}

type DeleteLedgerMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Deleted    bool                    `json:"deleted" yaml:"deleted"`
}

type UpdateLedgerMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, UpdateLedgerMetadataInput) (UpdateLedgerMetadataOutput, error)
}

type UpdateLedgerMetadataService struct {
	Handlers []UpdateLedgerMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteLedgerMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteLedgerMetadataInput) (DeleteLedgerMetadataOutput, error)
}

type DeleteLedgerMetadataService struct {
	Handlers []DeleteLedgerMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s UpdateLedgerMetadataService) Run(ctx context.Context, input UpdateLedgerMetadataInput) (UpdateLedgerMetadataOutput, error) {
	if input.Ledger == "" {
		return UpdateLedgerMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if len(input.Metadata) == 0 {
		return UpdateLedgerMetadataOutput{}, fmt.Errorf("metadata is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]UpdateLedgerMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return UpdateLedgerMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return UpdateLedgerMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return UpdateLedgerMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKUpdateLedgerMetadataHandlers(sdk *formance.Formance) []UpdateLedgerMetadataHandler {
	return []UpdateLedgerMetadataHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input UpdateLedgerMetadataInput) (UpdateLedgerMetadataOutput, error) {
				response, err := sdk.Ledger.V2.UpdateLedgerMetadata(ctx, operations.V2UpdateLedgerMetadataRequest{
					Ledger:      input.Ledger,
					RequestBody: input.Metadata,
				})
				if err != nil {
					return UpdateLedgerMetadataOutput{}, err
				}
				return UpdateLedgerMetadataOutput{Updated: response.StatusCode == 204}, nil
			},
		},
	}
}

func (s DeleteLedgerMetadataService) Run(ctx context.Context, input DeleteLedgerMetadataInput) (DeleteLedgerMetadataOutput, error) {
	if input.Ledger == "" {
		return DeleteLedgerMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Key == "" {
		return DeleteLedgerMetadataOutput{}, fmt.Errorf("metadata key is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeleteLedgerMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeleteLedgerMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeleteLedgerMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteLedgerMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKDeleteLedgerMetadataHandlers(sdk *formance.Formance) []DeleteLedgerMetadataHandler {
	return []DeleteLedgerMetadataHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input DeleteLedgerMetadataInput) (DeleteLedgerMetadataOutput, error) {
				response, err := sdk.Ledger.V2.DeleteLedgerMetadata(ctx, operations.V2DeleteLedgerMetadataRequest{
					Ledger: input.Ledger,
					Key:    input.Key,
				})
				if err != nil {
					return DeleteLedgerMetadataOutput{}, err
				}
				return DeleteLedgerMetadataOutput{Deleted: response.StatusCode == 204}, nil
			},
		},
	}
}
