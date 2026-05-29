package webhooks

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	ProductWebhooks capabilities.Product = "webhooks"

	FeatureActivateConfig     capabilities.Feature = "activateConfig"
	FeatureChangeConfigSecret capabilities.Feature = "changeConfigSecret"
	FeatureDeactivateConfig   capabilities.Feature = "deactivateConfig"
	FeatureDeleteConfig       capabilities.Feature = "deleteConfig"
	FeatureGetManyConfigs     capabilities.Feature = "getManyConfigs"
	FeatureInsertConfig       capabilities.Feature = "insertConfig"
)

type ConfigSummary struct {
	ID         string    `json:"id" yaml:"id"`
	Endpoint   string    `json:"endpoint" yaml:"endpoint"`
	EventTypes []string  `json:"eventTypes" yaml:"eventTypes"`
	Active     bool      `json:"active" yaml:"active"`
	CreatedAt  time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

type ConfigInput struct {
	ConfigID   string
	Endpoint   string
	EventTypes []string
	Name       *string
	Secret     *string
}

type ListConfigsInput struct {
	ConfigID string
	Endpoint string
}

type ListConfigsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Configs    []ConfigSummary         `json:"configs" yaml:"configs"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
}

type ConfigOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Config     ConfigSummary           `json:"config" yaml:"config"`
}

type ConfigIDInput struct {
	ConfigID string
}

type DeleteConfigOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ConfigID   string                  `json:"configID" yaml:"configID"`
}

type ChangeSecretInput struct {
	ConfigID string
	Secret   string
}

type ListConfigsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListConfigsInput) (ListConfigsOutput, error)
}

type ConfigHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ConfigInput) (ConfigOutput, error)
}

type ConfigIDHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ConfigIDInput) (ConfigOutput, error)
}

type DeleteConfigHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ConfigIDInput) (DeleteConfigOutput, error)
}

type ChangeSecretHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ChangeSecretInput) (ConfigOutput, error)
}

type ListConfigsService struct {
	Handlers []ListConfigsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type InsertConfigService struct {
	Handlers []ConfigHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ActivateConfigService struct {
	Handlers []ConfigIDHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeactivateConfigService struct {
	Handlers []ConfigIDHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteConfigService struct {
	Handlers []DeleteConfigHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ChangeSecretService struct {
	Handlers []ChangeSecretHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListConfigsService) Run(ctx context.Context, input ListConfigsInput) (ListConfigsOutput, error) {
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ListConfigsHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ListConfigsOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListConfigsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s InsertConfigService) Run(ctx context.Context, input ConfigInput) (ConfigOutput, error) {
	if input.Endpoint == "" {
		return ConfigOutput{}, fmt.Errorf("webhook endpoint is required")
	}
	if len(input.EventTypes) == 0 {
		return ConfigOutput{}, fmt.Errorf("at least one event type is required")
	}
	return runConfigService(ctx, input, s.Handlers, s.Resolve)
}

func (s ActivateConfigService) Run(ctx context.Context, input ConfigIDInput) (ConfigOutput, error) {
	return runConfigIDService(ctx, input, s.Handlers, s.Resolve)
}

func (s DeactivateConfigService) Run(ctx context.Context, input ConfigIDInput) (ConfigOutput, error) {
	return runConfigIDService(ctx, input, s.Handlers, s.Resolve)
}

func (s DeleteConfigService) Run(ctx context.Context, input ConfigIDInput) (DeleteConfigOutput, error) {
	if input.ConfigID == "" {
		return DeleteConfigOutput{}, fmt.Errorf("config id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler DeleteConfigHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return DeleteConfigOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteConfigOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ChangeSecretService) Run(ctx context.Context, input ChangeSecretInput) (ConfigOutput, error) {
	if input.ConfigID == "" {
		return ConfigOutput{}, fmt.Errorf("config id is required")
	}
	if input.Secret == "" {
		return ConfigOutput{}, fmt.Errorf("webhook secret is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ChangeSecretHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ConfigOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ConfigOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runConfigService(
	ctx context.Context,
	input ConfigInput,
	serviceHandlers []ConfigHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (ConfigOutput, error) {
	handler, selected, err := resolveHandler(ctx, serviceHandlers, resolve, func(handler ConfigHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ConfigOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ConfigOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runConfigIDService(
	ctx context.Context,
	input ConfigIDInput,
	serviceHandlers []ConfigIDHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (ConfigOutput, error) {
	if input.ConfigID == "" {
		return ConfigOutput{}, fmt.Errorf("config id is required")
	}
	handler, selected, err := resolveHandler(ctx, serviceHandlers, resolve, func(handler ConfigIDHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ConfigOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ConfigOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func resolveHandler[H any](
	ctx context.Context,
	serviceHandlers []H,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
	versionOf func(H) capabilities.APIVersion,
) (H, capabilities.APIVersion, error) {
	var zero H
	handlerVersions := make([]capabilities.APIVersion, 0, len(serviceHandlers))
	handlers := map[capabilities.APIVersion]H{}
	for _, handler := range serviceHandlers {
		version := versionOf(handler)
		handlerVersions = append(handlerVersions, version)
		handlers[version] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return zero, "", err
	}
	handler, ok := handlers[selected]
	if !ok {
		return zero, "", fmt.Errorf("resolved api version %s has no handler", selected)
	}
	return handler, selected, nil
}

func SDKListConfigsHandlers(sdk *formance.Formance) []ListConfigsHandler {
	return []ListConfigsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListConfigsInput) (ListConfigsOutput, error) {
				response, err := sdk.Webhooks.V1.GetManyConfigs(ctx, operations.GetManyConfigsRequest{
					ID:       optionalString(input.ConfigID),
					Endpoint: optionalString(input.Endpoint),
				})
				if err != nil {
					return ListConfigsOutput{}, err
				}
				if response.ConfigsResponse == nil {
					return ListConfigsOutput{}, fmt.Errorf("webhooks v1 list configs returned no data")
				}
				cursor := response.ConfigsResponse.Cursor
				configs := make([]ConfigSummary, 0, len(cursor.Data))
				for _, config := range cursor.Data {
					configs = append(configs, fromConfig(config))
				}
				return ListConfigsOutput{Configs: configs, HasMore: cursor.HasMore}, nil
			},
		},
	}
}

func SDKInsertConfigHandlers(sdk *formance.Formance) []ConfigHandler {
	return []ConfigHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ConfigInput) (ConfigOutput, error) {
				response, err := sdk.Webhooks.V1.InsertConfig(ctx, configUser(input))
				if err != nil {
					return ConfigOutput{}, err
				}
				if response.ConfigResponse == nil {
					return ConfigOutput{}, fmt.Errorf("webhooks v1 insert config returned no data")
				}
				return ConfigOutput{Config: fromConfig(response.ConfigResponse.Data)}, nil
			},
		},
	}
}

func SDKActivateConfigHandlers(sdk *formance.Formance) []ConfigIDHandler {
	return []ConfigIDHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ConfigIDInput) (ConfigOutput, error) {
				response, err := sdk.Webhooks.V1.ActivateConfig(ctx, operations.ActivateConfigRequest{ID: input.ConfigID})
				if err != nil {
					return ConfigOutput{}, err
				}
				if response.ConfigResponse == nil {
					return ConfigOutput{}, fmt.Errorf("webhooks v1 activate config returned no data")
				}
				return ConfigOutput{Config: fromConfig(response.ConfigResponse.Data)}, nil
			},
		},
	}
}

func SDKDeactivateConfigHandlers(sdk *formance.Formance) []ConfigIDHandler {
	return []ConfigIDHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ConfigIDInput) (ConfigOutput, error) {
				response, err := sdk.Webhooks.V1.DeactivateConfig(ctx, operations.DeactivateConfigRequest{ID: input.ConfigID})
				if err != nil {
					return ConfigOutput{}, err
				}
				if response.ConfigResponse == nil {
					return ConfigOutput{}, fmt.Errorf("webhooks v1 deactivate config returned no data")
				}
				return ConfigOutput{Config: fromConfig(response.ConfigResponse.Data)}, nil
			},
		},
	}
}

func SDKDeleteConfigHandlers(sdk *formance.Formance) []DeleteConfigHandler {
	return []DeleteConfigHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ConfigIDInput) (DeleteConfigOutput, error) {
				if _, err := sdk.Webhooks.V1.DeleteConfig(ctx, operations.DeleteConfigRequest{ID: input.ConfigID}); err != nil {
					return DeleteConfigOutput{}, err
				}
				return DeleteConfigOutput{ConfigID: input.ConfigID}, nil
			},
		},
	}
}

func SDKChangeSecretHandlers(sdk *formance.Formance) []ChangeSecretHandler {
	return []ChangeSecretHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ChangeSecretInput) (ConfigOutput, error) {
				response, err := sdk.Webhooks.V1.ChangeConfigSecret(ctx, operations.ChangeConfigSecretRequest{
					ID:                 input.ConfigID,
					ConfigChangeSecret: &shared.ConfigChangeSecret{Secret: input.Secret},
				})
				if err != nil {
					return ConfigOutput{}, err
				}
				if response.ConfigResponse == nil {
					return ConfigOutput{}, fmt.Errorf("webhooks v1 change config secret returned no data")
				}
				return ConfigOutput{Config: fromConfig(response.ConfigResponse.Data)}, nil
			},
		},
	}
}

func configUser(input ConfigInput) shared.ConfigUser {
	return shared.ConfigUser{
		Endpoint:   input.Endpoint,
		EventTypes: input.EventTypes,
		Name:       input.Name,
		Secret:     input.Secret,
	}
}

func fromConfig(config shared.WebhooksConfig) ConfigSummary {
	return ConfigSummary{
		ID:         config.ID,
		Endpoint:   config.Endpoint,
		EventTypes: config.EventTypes,
		Active:     config.Active,
		CreatedAt:  config.CreatedAt,
		UpdatedAt:  config.UpdatedAt,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
