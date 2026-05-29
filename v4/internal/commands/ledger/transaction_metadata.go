package ledger

import (
	"context"
	"fmt"
	"math/big"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type AddTransactionMetadataInput struct {
	Ledger        string
	TransactionID string
	Metadata      map[string]string
}

type AddTransactionMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Updated    bool                    `json:"updated" yaml:"updated"`
}

type DeleteTransactionMetadataInput struct {
	Ledger        string
	TransactionID string
	Key           string
}

type DeleteTransactionMetadataOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Deleted    bool                    `json:"deleted" yaml:"deleted"`
}

type AddTransactionMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, AddTransactionMetadataInput) (AddTransactionMetadataOutput, error)
}

type AddTransactionMetadataService struct {
	Handlers []AddTransactionMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteTransactionMetadataHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteTransactionMetadataInput) (DeleteTransactionMetadataOutput, error)
}

type DeleteTransactionMetadataService struct {
	Handlers []DeleteTransactionMetadataHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s AddTransactionMetadataService) Run(ctx context.Context, input AddTransactionMetadataInput) (AddTransactionMetadataOutput, error) {
	if input.Ledger == "" {
		return AddTransactionMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if input.TransactionID == "" {
		return AddTransactionMetadataOutput{}, fmt.Errorf("transaction id is required")
	}
	if len(input.Metadata) == 0 {
		return AddTransactionMetadataOutput{}, fmt.Errorf("metadata is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]AddTransactionMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return AddTransactionMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return AddTransactionMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return AddTransactionMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKAddTransactionMetadataHandlers(sdk *formance.Formance) []AddTransactionMetadataHandler {
	return []AddTransactionMetadataHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input AddTransactionMetadataInput) (AddTransactionMetadataOutput, error) {
				txid, err := parseTransactionID(input.TransactionID)
				if err != nil {
					return AddTransactionMetadataOutput{}, err
				}
				response, err := sdk.Ledger.V1.AddMetadataOnTransaction(ctx, operations.AddMetadataOnTransactionRequest{
					Ledger:      input.Ledger,
					Txid:        txid,
					RequestBody: stringMapToAny(input.Metadata),
				})
				if err != nil {
					return AddTransactionMetadataOutput{}, err
				}
				return AddTransactionMetadataOutput{Updated: response.StatusCode == 204}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input AddTransactionMetadataInput) (AddTransactionMetadataOutput, error) {
				txid, err := parseTransactionID(input.TransactionID)
				if err != nil {
					return AddTransactionMetadataOutput{}, err
				}
				response, err := sdk.Ledger.V2.AddMetadataOnTransaction(ctx, operations.V2AddMetadataOnTransactionRequest{
					Ledger:      input.Ledger,
					ID:          txid,
					RequestBody: input.Metadata,
				})
				if err != nil {
					return AddTransactionMetadataOutput{}, err
				}
				return AddTransactionMetadataOutput{Updated: response.StatusCode == 204}, nil
			},
		},
	}
}

func (s DeleteTransactionMetadataService) Run(ctx context.Context, input DeleteTransactionMetadataInput) (DeleteTransactionMetadataOutput, error) {
	if input.Ledger == "" {
		return DeleteTransactionMetadataOutput{}, fmt.Errorf("ledger is required")
	}
	if input.TransactionID == "" {
		return DeleteTransactionMetadataOutput{}, fmt.Errorf("transaction id is required")
	}
	if input.Key == "" {
		return DeleteTransactionMetadataOutput{}, fmt.Errorf("metadata key is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]DeleteTransactionMetadataHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return DeleteTransactionMetadataOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return DeleteTransactionMetadataOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteTransactionMetadataOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKDeleteTransactionMetadataHandlers(sdk *formance.Formance) []DeleteTransactionMetadataHandler {
	return []DeleteTransactionMetadataHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input DeleteTransactionMetadataInput) (DeleteTransactionMetadataOutput, error) {
				txid, err := parseTransactionID(input.TransactionID)
				if err != nil {
					return DeleteTransactionMetadataOutput{}, err
				}
				response, err := sdk.Ledger.V2.DeleteTransactionMetadata(ctx, operations.V2DeleteTransactionMetadataRequest{
					Ledger: input.Ledger,
					ID:     txid,
					Key:    input.Key,
				})
				if err != nil {
					return DeleteTransactionMetadataOutput{}, err
				}
				return DeleteTransactionMetadataOutput{Deleted: response.StatusCode == 204}, nil
			},
		},
	}
}

func parseTransactionID(value string) (*big.Int, error) {
	txid, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("transaction id must be an integer")
	}
	return txid, nil
}
