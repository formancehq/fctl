package ledger

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type AddAccountMetadataInput struct {
	Ledger   string
	Account  string
	Metadata map[string]string
}

type AddAccountMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Updated    bool                    `json:"updated" yaml:"updated"`
}

type DeleteAccountMetadataInput struct {
	Ledger  string
	Account string
	Key     string
}

type DeleteAccountMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Deleted    bool                    `json:"deleted" yaml:"deleted"`
}

type AddAccountMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, AddAccountMetadataInput) (AddAccountMetadataOutput, error)
}

type AddAccountMetadataService struct {
	Handlers []AddAccountMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteAccountMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteAccountMetadataInput) (DeleteAccountMetadataOutput, error)
}

type DeleteAccountMetadataService struct {
	Handlers []DeleteAccountMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s AddAccountMetadataService) Run(ctx context.Context, input AddAccountMetadataInput) (AddAccountMetadataOutput, error) {
	if input.Ledger == "" {
		return AddAccountMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Account == "" {
		return AddAccountMetadataOutput{}, fmt.Errorf("account is required")
	}
	if len(input.Metadata) == 0 {
		return AddAccountMetadataOutput{}, fmt.Errorf("metadata is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]AddAccountMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return AddAccountMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return AddAccountMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return AddAccountMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKAddAccountMetadataHandlers(sdk *formance.Formance) []AddAccountMetadataHandler {
	return []AddAccountMetadataHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input AddAccountMetadataInput) (AddAccountMetadataOutput, error) {
				response, err := sdk.Ledger.V1.AddMetadataToAccount(ctx, operations.AddMetadataToAccountRequest{
					Ledger:      input.Ledger,
					Address:     input.Account,
					RequestBody: stringMapToAny(input.Metadata),
				})
				if err != nil {
					return AddAccountMetadataOutput{}, err
				}
				return AddAccountMetadataOutput{Updated: response.StatusCode == 204}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input AddAccountMetadataInput) (AddAccountMetadataOutput, error) {
				response, err := sdk.Ledger.V2.AddMetadataToAccount(ctx, operations.V2AddMetadataToAccountRequest{
					Ledger:      input.Ledger,
					Address:     input.Account,
					RequestBody: input.Metadata,
				})
				if err != nil {
					return AddAccountMetadataOutput{}, err
				}
				return AddAccountMetadataOutput{Updated: response.StatusCode == 204}, nil
			},
		},
	}
}

func (s DeleteAccountMetadataService) Run(ctx context.Context, input DeleteAccountMetadataInput) (DeleteAccountMetadataOutput, error) {
	if input.Ledger == "" {
		return DeleteAccountMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Account == "" {
		return DeleteAccountMetadataOutput{}, fmt.Errorf("account is required")
	}
	if input.Key == "" {
		return DeleteAccountMetadataOutput{}, fmt.Errorf("metadata key is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeleteAccountMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeleteAccountMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeleteAccountMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteAccountMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKDeleteAccountMetadataHandlers(sdk *formance.Formance) []DeleteAccountMetadataHandler {
	return []DeleteAccountMetadataHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input DeleteAccountMetadataInput) (DeleteAccountMetadataOutput, error) {
				response, err := sdk.Ledger.V2.DeleteAccountMetadata(ctx, operations.V2DeleteAccountMetadataRequest{
					Ledger:  input.Ledger,
					Address: input.Account,
					Key:     input.Key,
				})
				if err != nil {
					return DeleteAccountMetadataOutput{}, err
				}
				return DeleteAccountMetadataOutput{Deleted: response.StatusCode == 204}, nil
			},
		},
	}
}
