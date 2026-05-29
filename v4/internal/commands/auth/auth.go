package auth

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	ProductAuth capabilities.Product = "auth"

	FeatureCreateClient capabilities.Feature = "createClient"
	FeatureCreateSecret capabilities.Feature = "createSecret"
	FeatureDeleteClient capabilities.Feature = "deleteClient"
	FeatureDeleteSecret capabilities.Feature = "deleteSecret"
	FeatureListClients  capabilities.Feature = "listClients"
	FeatureReadClient   capabilities.Feature = "readClient"
	FeatureListUsers    capabilities.Feature = "listUsers"
	FeatureReadUser     capabilities.Feature = "readUser"
	FeatureUpdateClient capabilities.Feature = "updateClient"
)

type ClientSecretSummary struct {
	ID         string            `json:"id" yaml:"id"`
	Name       string            `json:"name" yaml:"name"`
	LastDigits string            `json:"lastDigits" yaml:"lastDigits"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ClientSummary struct {
	ID                     string                `json:"id" yaml:"id"`
	Name                   string                `json:"name" yaml:"name"`
	Description            string                `json:"description,omitempty" yaml:"description,omitempty"`
	Metadata               map[string]string     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Scopes                 []string              `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	RedirectURIs           []string              `json:"redirectUris,omitempty" yaml:"redirectUris,omitempty"`
	PostLogoutRedirectURIs []string              `json:"postLogoutRedirectUris,omitempty" yaml:"postLogoutRedirectUris,omitempty"`
	Public                 *bool                 `json:"public,omitempty" yaml:"public,omitempty"`
	Trusted                *bool                 `json:"trusted,omitempty" yaml:"trusted,omitempty"`
	Secrets                []ClientSecretSummary `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

type UserSummary struct {
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Email   string `json:"email,omitempty" yaml:"email,omitempty"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
}

type ListClientsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Clients    []ClientSummary         `json:"clients" yaml:"clients"`
}

type ClientMutationInput struct {
	ClientID               string
	Name                   string
	Description            *string
	Metadata               map[string]string
	Scopes                 []string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	Public                 *bool
	Trusted                *bool
}

type ClientMutationOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Client     ClientSummary           `json:"client" yaml:"client"`
}

type GetClientInput struct {
	ClientID string
}

type GetClientOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Client     ClientSummary           `json:"client" yaml:"client"`
}

type DeleteClientInput struct {
	ClientID string
}

type DeleteClientOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ClientID   string                  `json:"clientID" yaml:"clientID"`
}

type CreateSecretInput struct {
	ClientID string
	Name     string
	Metadata map[string]string
}

type SecretSummary struct {
	ID         string            `json:"id" yaml:"id"`
	Name       string            `json:"name" yaml:"name"`
	LastDigits string            `json:"lastDigits" yaml:"lastDigits"`
	Clear      string            `json:"clear,omitempty" yaml:"clear,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type CreateSecretOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ClientID   string                  `json:"clientID" yaml:"clientID"`
	Secret     SecretSummary           `json:"secret" yaml:"secret"`
}

type DeleteSecretInput struct {
	ClientID string
	SecretID string
}

type DeleteSecretOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	ClientID   string                  `json:"clientID" yaml:"clientID"`
	SecretID   string                  `json:"secretID" yaml:"secretID"`
}

type ListUsersOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Users      []UserSummary           `json:"users" yaml:"users"`
}

type GetUserInput struct {
	UserID string
}

type GetUserOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	User       UserSummary             `json:"user" yaml:"user"`
}

type ListClientsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context) (ListClientsOutput, error)
}

type ClientMutationHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ClientMutationInput) (ClientMutationOutput, error)
}

type GetClientHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetClientInput) (GetClientOutput, error)
}

type DeleteClientHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteClientInput) (DeleteClientOutput, error)
}

type CreateSecretHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreateSecretInput) (CreateSecretOutput, error)
}

type DeleteSecretHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeleteSecretInput) (DeleteSecretOutput, error)
}

type ListUsersHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context) (ListUsersOutput, error)
}

type GetUserHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetUserInput) (GetUserOutput, error)
}

type ListClientsService struct {
	Handlers []ListClientsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateClientService struct {
	Handlers []ClientMutationHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetClientService struct {
	Handlers []GetClientHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type UpdateClientService struct {
	Handlers []ClientMutationHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteClientService struct {
	Handlers []DeleteClientHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreateSecretService struct {
	Handlers []CreateSecretHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeleteSecretService struct {
	Handlers []DeleteSecretHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListUsersService struct {
	Handlers []ListUsersHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetUserService struct {
	Handlers []GetUserHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListClientsService) Run(ctx context.Context) (ListClientsOutput, error) {
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ListClientsHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ListClientsOutput{}, err
	}
	output, err := handler.Run(ctx)
	if err != nil {
		return ListClientsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreateClientService) Run(ctx context.Context, input ClientMutationInput) (ClientMutationOutput, error) {
	if input.Name == "" {
		return ClientMutationOutput{}, fmt.Errorf("client name is required")
	}
	return runClientMutationService(ctx, input, s.Handlers, s.Resolve)
}

func (s GetClientService) Run(ctx context.Context, input GetClientInput) (GetClientOutput, error) {
	if input.ClientID == "" {
		return GetClientOutput{}, fmt.Errorf("client id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler GetClientHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return GetClientOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetClientOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s UpdateClientService) Run(ctx context.Context, input ClientMutationInput) (ClientMutationOutput, error) {
	if input.ClientID == "" {
		return ClientMutationOutput{}, fmt.Errorf("client id is required")
	}
	if input.Name == "" {
		return ClientMutationOutput{}, fmt.Errorf("client name is required")
	}
	return runClientMutationService(ctx, input, s.Handlers, s.Resolve)
}

func (s DeleteClientService) Run(ctx context.Context, input DeleteClientInput) (DeleteClientOutput, error) {
	if input.ClientID == "" {
		return DeleteClientOutput{}, fmt.Errorf("client id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler DeleteClientHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return DeleteClientOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteClientOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreateSecretService) Run(ctx context.Context, input CreateSecretInput) (CreateSecretOutput, error) {
	if input.ClientID == "" {
		return CreateSecretOutput{}, fmt.Errorf("client id is required")
	}
	if input.Name == "" {
		return CreateSecretOutput{}, fmt.Errorf("secret name is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler CreateSecretHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return CreateSecretOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreateSecretOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s DeleteSecretService) Run(ctx context.Context, input DeleteSecretInput) (DeleteSecretOutput, error) {
	if input.ClientID == "" {
		return DeleteSecretOutput{}, fmt.Errorf("client id is required")
	}
	if input.SecretID == "" {
		return DeleteSecretOutput{}, fmt.Errorf("secret id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler DeleteSecretHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return DeleteSecretOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeleteSecretOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ListUsersService) Run(ctx context.Context) (ListUsersOutput, error) {
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ListUsersHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ListUsersOutput{}, err
	}
	output, err := handler.Run(ctx)
	if err != nil {
		return ListUsersOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetUserService) Run(ctx context.Context, input GetUserInput) (GetUserOutput, error) {
	if input.UserID == "" {
		return GetUserOutput{}, fmt.Errorf("user id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler GetUserHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return GetUserOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetUserOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func runClientMutationService(
	ctx context.Context,
	input ClientMutationInput,
	serviceHandlers []ClientMutationHandler,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
) (ClientMutationOutput, error) {
	handler, selected, err := resolveHandler(ctx, serviceHandlers, resolve, func(handler ClientMutationHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ClientMutationOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ClientMutationOutput{}, err
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

func SDKCreateClientHandlers(sdk *formance.Formance) []ClientMutationHandler {
	return []ClientMutationHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ClientMutationInput) (ClientMutationOutput, error) {
				response, err := sdk.Auth.V1.CreateClient(ctx, clientRequest(input))
				if err != nil {
					return ClientMutationOutput{}, err
				}
				if response.CreateClientResponse == nil || response.CreateClientResponse.Data == nil {
					return ClientMutationOutput{}, fmt.Errorf("auth v1 create client returned no data")
				}
				return ClientMutationOutput{Client: fromClient(*response.CreateClientResponse.Data)}, nil
			},
		},
	}
}

func SDKListClientsHandlers(sdk *formance.Formance) []ListClientsHandler {
	return []ListClientsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context) (ListClientsOutput, error) {
				response, err := sdk.Auth.V1.ListClients(ctx)
				if err != nil {
					return ListClientsOutput{}, err
				}
				if response.ListClientsResponse == nil {
					return ListClientsOutput{}, fmt.Errorf("auth v1 list clients returned no data")
				}
				clients := make([]ClientSummary, 0, len(response.ListClientsResponse.Data))
				for _, client := range response.ListClientsResponse.Data {
					clients = append(clients, fromClient(client))
				}
				return ListClientsOutput{Clients: clients}, nil
			},
		},
	}
}

func SDKGetClientHandlers(sdk *formance.Formance) []GetClientHandler {
	return []GetClientHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetClientInput) (GetClientOutput, error) {
				response, err := sdk.Auth.V1.ReadClient(ctx, operations.ReadClientRequest{ClientID: input.ClientID})
				if err != nil {
					return GetClientOutput{}, err
				}
				if response.ReadClientResponse == nil || response.ReadClientResponse.Data == nil {
					return GetClientOutput{}, fmt.Errorf("auth v1 read client returned no data")
				}
				return GetClientOutput{Client: fromClient(*response.ReadClientResponse.Data)}, nil
			},
		},
	}
}

func SDKUpdateClientHandlers(sdk *formance.Formance) []ClientMutationHandler {
	return []ClientMutationHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ClientMutationInput) (ClientMutationOutput, error) {
				response, err := sdk.Auth.V1.UpdateClient(ctx, operations.UpdateClientRequest{
					ClientID:            input.ClientID,
					CreateClientRequest: clientRequest(input),
				})
				if err != nil {
					return ClientMutationOutput{}, err
				}
				if response.UpdateClientResponse == nil || response.UpdateClientResponse.Data == nil {
					return ClientMutationOutput{}, fmt.Errorf("auth v1 update client returned no data")
				}
				return ClientMutationOutput{Client: fromClient(*response.UpdateClientResponse.Data)}, nil
			},
		},
	}
}

func SDKDeleteClientHandlers(sdk *formance.Formance) []DeleteClientHandler {
	return []DeleteClientHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeleteClientInput) (DeleteClientOutput, error) {
				if _, err := sdk.Auth.V1.DeleteClient(ctx, operations.DeleteClientRequest{ClientID: input.ClientID}); err != nil {
					return DeleteClientOutput{}, err
				}
				return DeleteClientOutput{ClientID: input.ClientID}, nil
			},
		},
	}
}

func SDKCreateSecretHandlers(sdk *formance.Formance) []CreateSecretHandler {
	return []CreateSecretHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreateSecretInput) (CreateSecretOutput, error) {
				response, err := sdk.Auth.V1.CreateSecret(ctx, operations.CreateSecretRequest{
					ClientID: input.ClientID,
					CreateSecretRequest: &shared.CreateSecretRequest{
						Name:     input.Name,
						Metadata: input.Metadata,
					},
				})
				if err != nil {
					return CreateSecretOutput{}, err
				}
				if response.CreateSecretResponse == nil || response.CreateSecretResponse.Data == nil {
					return CreateSecretOutput{}, fmt.Errorf("auth v1 create secret returned no data")
				}
				return CreateSecretOutput{
					ClientID: input.ClientID,
					Secret:   fromSecret(*response.CreateSecretResponse.Data),
				}, nil
			},
		},
	}
}

func SDKDeleteSecretHandlers(sdk *formance.Formance) []DeleteSecretHandler {
	return []DeleteSecretHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeleteSecretInput) (DeleteSecretOutput, error) {
				if _, err := sdk.Auth.V1.DeleteSecret(ctx, operations.DeleteSecretRequest{
					ClientID: input.ClientID,
					SecretID: input.SecretID,
				}); err != nil {
					return DeleteSecretOutput{}, err
				}
				return DeleteSecretOutput{ClientID: input.ClientID, SecretID: input.SecretID}, nil
			},
		},
	}
}

func SDKListUsersHandlers(sdk *formance.Formance) []ListUsersHandler {
	return []ListUsersHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context) (ListUsersOutput, error) {
				response, err := sdk.Auth.V1.ListUsers(ctx)
				if err != nil {
					return ListUsersOutput{}, err
				}
				if response.ListUsersResponse == nil {
					return ListUsersOutput{}, fmt.Errorf("auth v1 list users returned no data")
				}
				users := make([]UserSummary, 0, len(response.ListUsersResponse.Data))
				for _, user := range response.ListUsersResponse.Data {
					users = append(users, fromUser(user))
				}
				return ListUsersOutput{Users: users}, nil
			},
		},
	}
}

func SDKGetUserHandlers(sdk *formance.Formance) []GetUserHandler {
	return []GetUserHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetUserInput) (GetUserOutput, error) {
				response, err := sdk.Auth.V1.ReadUser(ctx, operations.ReadUserRequest{UserID: input.UserID})
				if err != nil {
					return GetUserOutput{}, err
				}
				if response.ReadUserResponse == nil || response.ReadUserResponse.Data == nil {
					return GetUserOutput{}, fmt.Errorf("auth v1 read user returned no data")
				}
				return GetUserOutput{User: fromUser(*response.ReadUserResponse.Data)}, nil
			},
		},
	}
}

func clientRequest(input ClientMutationInput) *shared.CreateClientRequest {
	return &shared.CreateClientRequest{
		Description:            input.Description,
		Metadata:               input.Metadata,
		Name:                   input.Name,
		PostLogoutRedirectUris: input.PostLogoutRedirectURIs,
		Public:                 input.Public,
		RedirectUris:           input.RedirectURIs,
		Scopes:                 input.Scopes,
		Trusted:                input.Trusted,
	}
}

func fromClient(client shared.Client) ClientSummary {
	description := ""
	if client.Description != nil {
		description = *client.Description
	}
	secrets := make([]ClientSecretSummary, 0, len(client.Secrets))
	for _, secret := range client.Secrets {
		secrets = append(secrets, ClientSecretSummary{
			ID:         secret.ID,
			Name:       secret.Name,
			LastDigits: secret.LastDigits,
			Metadata:   secret.Metadata,
		})
	}
	return ClientSummary{
		ID:                     client.ID,
		Name:                   client.Name,
		Description:            description,
		Metadata:               client.Metadata,
		Scopes:                 client.Scopes,
		RedirectURIs:           client.RedirectUris,
		PostLogoutRedirectURIs: client.PostLogoutRedirectUris,
		Public:                 client.Public,
		Trusted:                client.Trusted,
		Secrets:                secrets,
	}
}

func fromSecret(secret shared.Secret) SecretSummary {
	return SecretSummary{
		ID:         secret.ID,
		Name:       secret.Name,
		LastDigits: secret.LastDigits,
		Clear:      secret.Clear,
		Metadata:   secret.Metadata,
	}
}

func fromUser(user shared.User) UserSummary {
	return UserSummary{
		ID:      stringValue(user.ID),
		Email:   stringValue(user.Email),
		Subject: stringValue(user.Subject),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
