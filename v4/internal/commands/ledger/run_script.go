package ledger

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type RunScriptInput struct {
	Ledger      string
	Script      string
	AccountVars map[string]string
	AmountVars  map[string]string
	PortionVars map[string]string
	Metadata    map[string]string
	Reference   string
	Timestamp   *time.Time
}

type RunScriptOutput struct {
	APIVersion  capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Transaction TransactionSummary      `json:"transaction" yaml:"transaction"`
}

type RunScriptHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, RunScriptInput) (RunScriptOutput, error)
}

type RunScriptService struct {
	Handlers []RunScriptHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s RunScriptService) Run(ctx context.Context, input RunScriptInput) (RunScriptOutput, error) {
	if input.Ledger == "" {
		return RunScriptOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Script == "" {
		return RunScriptOutput{}, fmt.Errorf("script is required")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]RunScriptHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return RunScriptOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return RunScriptOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return RunScriptOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKRunScriptHandlers(sdk *formance.Formance) []RunScriptHandler {
	return []RunScriptHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input RunScriptInput) (RunScriptOutput, error) {
				vars, err := toV1ScriptVars(input)
				if err != nil {
					return RunScriptOutput{}, err
				}
				response, err := sdk.Ledger.V1.CreateTransaction(ctx, operations.CreateTransactionRequest{
					Ledger: input.Ledger,
					PostTransaction: shared.PostTransaction{
						Metadata:  stringMapToAny(input.Metadata),
						Reference: optionalString(input.Reference),
						Script: &shared.PostTransactionScript{
							Plain: input.Script,
							Vars:  vars,
						},
						Timestamp: input.Timestamp,
					},
				})
				if err != nil {
					return RunScriptOutput{}, err
				}
				if response.TransactionsResponse == nil || len(response.TransactionsResponse.Data) == 0 {
					return RunScriptOutput{}, fmt.Errorf("ledger v1 run script returned no data")
				}
				return RunScriptOutput{Transaction: fromV1Transaction(response.TransactionsResponse.Data[0])}, nil
			},
		},
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input RunScriptInput) (RunScriptOutput, error) {
				vars := toV2ScriptVars(input)
				response, err := sdk.Ledger.V2.CreateTransaction(ctx, operations.V2CreateTransactionRequest{
					Ledger: input.Ledger,
					V2PostTransaction: shared.V2PostTransaction{
						Metadata:  input.Metadata,
						Reference: optionalString(input.Reference),
						Script: &shared.V2PostTransactionScript{
							Plain: &input.Script,
							Vars:  vars,
						},
						Timestamp: input.Timestamp,
					},
				})
				if err != nil {
					return RunScriptOutput{}, err
				}
				if response.V2CreateTransactionResponse == nil {
					return RunScriptOutput{}, fmt.Errorf("ledger v2 run script returned no data")
				}
				return RunScriptOutput{Transaction: fromV2Transaction(response.V2CreateTransactionResponse.Data)}, nil
			},
		},
	}
}

func toV1ScriptVars(input RunScriptInput) (map[string]any, error) {
	vars := map[string]any{}
	for key, value := range input.AccountVars {
		vars[key] = value
	}
	for key, value := range input.PortionVars {
		vars[key] = value
	}
	for key, value := range input.AmountVars {
		amount, asset, err := splitAmountAsset(value)
		if err != nil {
			return nil, fmt.Errorf("amount var %s: %w", key, err)
		}
		vars[key] = map[string]any{"amount": amount, "asset": asset}
	}
	if len(vars) == 0 {
		return nil, nil
	}
	return vars, nil
}

func toV2ScriptVars(input RunScriptInput) map[string]string {
	vars := map[string]string{}
	for key, value := range input.AccountVars {
		vars[key] = value
	}
	for key, value := range input.PortionVars {
		vars[key] = value
	}
	for key, value := range input.AmountVars {
		vars[key] = value
	}
	if len(vars) == 0 {
		return nil
	}
	return vars
}

func splitAmountAsset(value string) (any, string, error) {
	amount, asset, ok := stringsCut(value, "/")
	if !ok || amount == "" || asset == "" {
		return nil, "", fmt.Errorf("must use amount/asset format")
	}
	parsed, err := parseAmount(amount)
	if err != nil {
		return nil, "", err
	}
	return parsed, asset, nil
}

func stringsCut(value string, sep string) (string, string, bool) {
	for i := 0; i+len(sep) <= len(value); i++ {
		if value[i:i+len(sep)] == sep {
			return value[:i], value[i+len(sep):], true
		}
	}
	return value, "", false
}
