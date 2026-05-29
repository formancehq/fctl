package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	FeatureGetSchema    capabilities.Feature = "getSchema"
	FeatureInsertSchema capabilities.Feature = "insertSchema"
	FeatureListSchemas  capabilities.Feature = "listSchemas"
)

type ListSchemasInput struct {
	Ledger   string
	PageSize int64
	Cursor   string
}

type ListSchemasOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Schemas    []SchemaSummary         `json:"schemas" yaml:"schemas"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type SchemaSummary struct {
	Version           string    `json:"version" yaml:"version"`
	CreatedAt         time.Time `json:"createdAt" yaml:"createdAt"`
	ChartSegments     int       `json:"chartSegments" yaml:"chartSegments"`
	QueryTemplates    int       `json:"queryTemplates" yaml:"queryTemplates"`
	TransactionModels int       `json:"transactionModels" yaml:"transactionModels"`
}

type GetSchemaInput struct {
	Ledger  string
	Version string
}

type GetSchemaOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Schema     shared.V2Schema         `json:"schema" yaml:"schema"`
}

type InsertSchemaInput struct {
	Ledger         string
	Version        string
	Data           []byte
	IdempotencyKey string
}

type InsertSchemaOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Ledger     string                  `json:"ledger" yaml:"ledger"`
	Version    string                  `json:"version" yaml:"version"`
	Inserted   bool                    `json:"inserted" yaml:"inserted"`
}

type ListSchemasHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListSchemasInput) (ListSchemasOutput, error)
}

type GetSchemaHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetSchemaInput) (GetSchemaOutput, error)
}

type InsertSchemaHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, InsertSchemaInput) (InsertSchemaOutput, error)
}

type ListSchemasService struct {
	Handlers []ListSchemasHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetSchemaService struct {
	Handlers []GetSchemaHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type InsertSchemaService struct {
	Handlers []InsertSchemaHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListSchemasService) Run(ctx context.Context, input ListSchemasInput) (ListSchemasOutput, error) {
	if input.Ledger == "" {
		return ListSchemasOutput{}, fmt.Errorf("ledger is required")
	}
	return runListSchemas(ctx, s, input)
}

func (s GetSchemaService) Run(ctx context.Context, input GetSchemaInput) (GetSchemaOutput, error) {
	if input.Ledger == "" {
		return GetSchemaOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Version == "" {
		return GetSchemaOutput{}, fmt.Errorf("schema version is required")
	}
	return runGetSchema(ctx, s, input)
}

func (s InsertSchemaService) Run(ctx context.Context, input InsertSchemaInput) (InsertSchemaOutput, error) {
	if input.Ledger == "" {
		return InsertSchemaOutput{}, fmt.Errorf("ledger is required")
	}
	if input.Version == "" {
		return InsertSchemaOutput{}, fmt.Errorf("schema version is required")
	}
	if len(input.Data) == 0 {
		return InsertSchemaOutput{}, fmt.Errorf("schema data is required")
	}
	return runInsertSchema(ctx, s, input)
}

func SDKListSchemasHandlers(sdk *formance.Formance) []ListSchemasHandler {
	return []ListSchemasHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ListSchemasInput) (ListSchemasOutput, error) {
				request := operations.V2ListSchemasRequest{
					Ledger:   input.Ledger,
					PageSize: pointer(input.PageSize),
				}
				if input.Cursor != "" {
					request.Cursor = pointer(input.Cursor)
				}
				response, err := sdk.Ledger.V2.ListSchemas(ctx, request)
				if err != nil {
					return ListSchemasOutput{}, err
				}
				if response.V2SchemasCursorResponse == nil {
					return ListSchemasOutput{}, fmt.Errorf("ledger v2 list schemas returned no cursor")
				}
				return fromV2SchemasCursor(response.V2SchemasCursorResponse.Cursor), nil
			},
		},
	}
}

func SDKGetSchemaHandlers(sdk *formance.Formance) []GetSchemaHandler {
	return []GetSchemaHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input GetSchemaInput) (GetSchemaOutput, error) {
				response, err := sdk.Ledger.V2.GetSchema(ctx, operations.V2GetSchemaRequest{
					Ledger:  input.Ledger,
					Version: input.Version,
				})
				if err != nil {
					return GetSchemaOutput{}, err
				}
				if response.V2SchemaResponse == nil {
					return GetSchemaOutput{}, fmt.Errorf("ledger v2 get schema returned no schema")
				}
				return GetSchemaOutput{Schema: response.V2SchemaResponse.Data}, nil
			},
		},
	}
}

func SDKInsertSchemaHandlers(sdk *formance.Formance) []InsertSchemaHandler {
	return []InsertSchemaHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input InsertSchemaInput) (InsertSchemaOutput, error) {
				var data shared.V2SchemaData
				if err := json.Unmarshal(input.Data, &data); err != nil {
					return InsertSchemaOutput{}, err
				}
				request := operations.V2InsertSchemaRequest{
					V2SchemaData: data,
					Ledger:       input.Ledger,
					Version:      input.Version,
				}
				if input.IdempotencyKey != "" {
					request.IdempotencyKey = pointer(input.IdempotencyKey)
				}
				if _, err := sdk.Ledger.V2.InsertSchema(ctx, request); err != nil {
					return InsertSchemaOutput{}, err
				}
				return InsertSchemaOutput{Ledger: input.Ledger, Version: input.Version, Inserted: true}, nil
			},
		},
	}
}

func runListSchemas(ctx context.Context, service ListSchemasService, input ListSchemasInput) (ListSchemasOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(service.Handlers))
	handlers := map[capabilities.APIVersion]ListSchemasHandler{}
	for _, handler := range service.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := service.Resolve(ctx, handlerVersions)
	if err != nil {
		return ListSchemasOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ListSchemasOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListSchemasOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runGetSchema(ctx context.Context, service GetSchemaService, input GetSchemaInput) (GetSchemaOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(service.Handlers))
	handlers := map[capabilities.APIVersion]GetSchemaHandler{}
	for _, handler := range service.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := service.Resolve(ctx, handlerVersions)
	if err != nil {
		return GetSchemaOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return GetSchemaOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetSchemaOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runInsertSchema(ctx context.Context, service InsertSchemaService, input InsertSchemaInput) (InsertSchemaOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(service.Handlers))
	handlers := map[capabilities.APIVersion]InsertSchemaHandler{}
	for _, handler := range service.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}
	selected, err := service.Resolve(ctx, handlerVersions)
	if err != nil {
		return InsertSchemaOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return InsertSchemaOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return InsertSchemaOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func fromV2SchemasCursor(cursor shared.V2SchemasCursor) ListSchemasOutput {
	schemas := make([]SchemaSummary, 0, len(cursor.Data))
	for _, schema := range cursor.Data {
		schemas = append(schemas, fromV2SchemaSummary(schema))
	}
	return ListSchemasOutput{
		Schemas:  schemas,
		HasMore:  cursor.HasMore,
		PageSize: cursor.PageSize,
		Next:     cursor.Next,
		Previous: cursor.Previous,
	}
}

func fromV2SchemaSummary(schema shared.V2Schema) SchemaSummary {
	return SchemaSummary{
		Version:           schema.Version,
		CreatedAt:         schema.CreatedAt,
		ChartSegments:     len(schema.Chart),
		QueryTemplates:    len(schema.Queries),
		TransactionModels: len(schema.Transactions),
	}
}
