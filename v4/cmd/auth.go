package cmd

import (
	"context"
	"fmt"
	"strings"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	authcmd "github.com/formancehq/fctl/v4/internal/commands/auth"
)

func newAuthCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "Manage Auth service resources and CLI sessions",
	}
	command.AddCommand(newAuthClientsCommand())
	command.AddCommand(newAuthUsersCommand())
	return command
}

func newAuthClientsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "clients",
		Short: "Manage Auth clients",
	}
	command.AddCommand(newAuthClientsCreateCommand())
	command.AddCommand(newAuthClientsListCommand())
	command.AddCommand(newAuthClientsShowCommand("show", []string{"sh"}, false))
	command.AddCommand(newAuthClientsShowCommand("get", nil, true))
	command.AddCommand(newAuthClientsUpdateCommand())
	command.AddCommand(newAuthClientsDeleteCommand())
	command.AddCommand(newAuthClientsSecretsCommand())
	return command
}

func newAuthClientsSecretsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "secrets",
		Short: "Manage Auth client secrets",
	}
	command.AddCommand(newAuthClientsSecretsCreateCommand())
	command.AddCommand(newAuthClientsSecretsDeleteCommand())
	return command
}

func newAuthClientsCreateCommand() *cobra.Command {
	var input authClientFlags
	var apiVersion string

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an Auth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := input.mutationInput(cmd, "", args[0])
			if err != nil {
				return err
			}
			service, err := newCreateAuthClientService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), request)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthClientMutated(cmd, output, "created")
		},
	}
	bindAuthClientFlags(command, &input, false)
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthUsersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "users",
		Short: "Manage Auth users",
	}
	command.AddCommand(newAuthUsersListCommand())
	command.AddCommand(newAuthUsersShowCommand("show", []string{"sh"}, false))
	command.AddCommand(newAuthUsersShowCommand("get", nil, true))
	return command
}

func newAuthClientsListCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Auth clients",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := newListAuthClientsService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context())
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthClients(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthClientsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <client-id>",
		Aliases: aliases,
		Short:   "Show an Auth client",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command auth clients get has been deprecated, use auth clients show")
			}
			service, err := newGetAuthClientService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), authcmd.GetClientInput{ClientID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthClient(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use auth clients show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthClientsUpdateCommand() *cobra.Command {
	var input authClientFlags
	var apiVersion string

	command := &cobra.Command{
		Use:   "update <client-id>",
		Short: "Update an Auth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := input.mutationInput(cmd, args[0], input.name)
			if err != nil {
				return err
			}
			service, err := newUpdateAuthClientService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), request)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthClientMutated(cmd, output, "updated")
		},
	}
	bindAuthClientFlags(command, &input, true)
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthClientsDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <client-id>",
		Short: "Delete an Auth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("auth clients delete requires --confirm")
			}
			service, err := newDeleteAuthClientService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), authcmd.DeleteClientInput{ClientID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthClientDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm client deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthClientsSecretsCreateCommand() *cobra.Command {
	var metadata []string
	var apiVersion string

	command := &cobra.Command{
		Use:   "create <client-id> <secret-name>",
		Short: "Create an Auth client secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			service, err := newCreateAuthSecretService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), authcmd.CreateSecretInput{
				ClientID: args[0],
				Name:     args[1],
				Metadata: parsedMetadata,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthSecretCreated(cmd, output)
		},
	}
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Secret metadata as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthClientsSecretsDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <client-id> <secret-id>",
		Short: "Delete an Auth client secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("auth clients secrets delete requires --confirm")
			}
			service, err := newDeleteAuthSecretService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), authcmd.DeleteSecretInput{ClientID: args[0], SecretID: args[1]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthSecretDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm secret deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthUsersListCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Auth users",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := newListAuthUsersService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context())
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthUsers(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

func newAuthUsersShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <user-id>",
		Aliases: aliases,
		Short:   "Show an Auth user",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command auth users get has been deprecated, use auth users show")
			}
			service, err := newGetAuthUserService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), authcmd.GetUserInput{UserID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderAuthUser(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use auth users show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin auth API version")
	return command
}

type authClientFlags struct {
	name                   string
	description            string
	metadata               []string
	scopes                 []string
	redirectURIs           []string
	postLogoutRedirectURIs []string
	public                 bool
	trusted                bool
}

func bindAuthClientFlags(command *cobra.Command, input *authClientFlags, includeName bool) {
	if includeName {
		command.Flags().StringVar(&input.name, "name", "", "Client name")
	}
	command.Flags().StringVar(&input.description, "description", "", "Client description")
	command.Flags().StringArrayVar(&input.metadata, "metadata", nil, "Client metadata as key=value")
	command.Flags().StringArrayVar(&input.scopes, "scope", nil, "OAuth scope")
	command.Flags().StringArrayVar(&input.redirectURIs, "redirect-uri", nil, "OAuth redirect URI")
	command.Flags().StringArrayVar(&input.postLogoutRedirectURIs, "post-logout-redirect-uri", nil, "Post logout redirect URI")
	command.Flags().BoolVar(&input.public, "public", false, "Mark client as public")
	command.Flags().BoolVar(&input.trusted, "trusted", false, "Mark client as trusted")
}

func (f authClientFlags) mutationInput(cmd *cobra.Command, clientID string, name string) (authcmd.ClientMutationInput, error) {
	metadata, err := parseMetadataFlags(f.metadata)
	if err != nil {
		return authcmd.ClientMutationInput{}, err
	}
	var description *string
	if cmd.Flags().Changed("description") {
		description = &f.description
	}
	var public *bool
	if cmd.Flags().Changed("public") {
		public = &f.public
	}
	var trusted *bool
	if cmd.Flags().Changed("trusted") {
		trusted = &f.trusted
	}
	return authcmd.ClientMutationInput{
		ClientID:               clientID,
		Name:                   name,
		Description:            description,
		Metadata:               metadata,
		Scopes:                 f.scopes,
		RedirectURIs:           f.redirectURIs,
		PostLogoutRedirectURIs: f.postLogoutRedirectURIs,
		Public:                 public,
		Trusted:                trusted,
	}, nil
}

func newCreateAuthClientService(cmd *cobra.Command, apiVersion string) (authcmd.CreateClientService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.CreateClientService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.CreateClientService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.CreateClientService{
		Handlers: authcmd.SDKCreateClientHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureCreateClient, apiVersion),
	}, nil
}

func newListAuthClientsService(cmd *cobra.Command, apiVersion string) (authcmd.ListClientsService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.ListClientsService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.ListClientsService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.ListClientsService{
		Handlers: authcmd.SDKListClientsHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureListClients, apiVersion),
	}, nil
}

func newGetAuthClientService(cmd *cobra.Command, apiVersion string) (authcmd.GetClientService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.GetClientService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.GetClientService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.GetClientService{
		Handlers: authcmd.SDKGetClientHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureReadClient, apiVersion),
	}, nil
}

func newUpdateAuthClientService(cmd *cobra.Command, apiVersion string) (authcmd.UpdateClientService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.UpdateClientService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.UpdateClientService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.UpdateClientService{
		Handlers: authcmd.SDKUpdateClientHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureUpdateClient, apiVersion),
	}, nil
}

func newDeleteAuthClientService(cmd *cobra.Command, apiVersion string) (authcmd.DeleteClientService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.DeleteClientService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.DeleteClientService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.DeleteClientService{
		Handlers: authcmd.SDKDeleteClientHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureDeleteClient, apiVersion),
	}, nil
}

func newCreateAuthSecretService(cmd *cobra.Command, apiVersion string) (authcmd.CreateSecretService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.CreateSecretService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.CreateSecretService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.CreateSecretService{
		Handlers: authcmd.SDKCreateSecretHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureCreateSecret, apiVersion),
	}, nil
}

func newDeleteAuthSecretService(cmd *cobra.Command, apiVersion string) (authcmd.DeleteSecretService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.DeleteSecretService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.DeleteSecretService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.DeleteSecretService{
		Handlers: authcmd.SDKDeleteSecretHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureDeleteSecret, apiVersion),
	}, nil
}

func newListAuthUsersService(cmd *cobra.Command, apiVersion string) (authcmd.ListUsersService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.ListUsersService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.ListUsersService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.ListUsersService{
		Handlers: authcmd.SDKListUsersHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureListUsers, apiVersion),
	}, nil
}

func newGetAuthUserService(cmd *cobra.Command, apiVersion string) (authcmd.GetUserService, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return authcmd.GetUserService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return authcmd.GetUserService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return authcmd.GetUserService{
		Handlers: authcmd.SDKGetUserHandlers(sdk),
		Resolve:  authVersionResolver(rt, authcmd.FeatureReadUser, apiVersion),
	}, nil
}

func authVersionResolver(rt interface {
	ResolveAPIVersion(context.Context, capabilities.VersionResolutionRequest) (capabilities.APIVersion, error)
}, feature capabilities.Feature, apiVersion string) func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
	return func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
		request := capabilities.VersionResolutionRequest{
			Product:         authcmd.ProductAuth,
			Feature:         feature,
			HandlerVersions: handlerVersions,
		}
		if apiVersion != "" {
			request.Policy = capabilities.VersionPolicyPinned
			request.PinnedVersion = capabilities.APIVersion(apiVersion)
		}
		return rt.ResolveAPIVersion(ctx, request)
	}
}

func renderAuthClients(cmd *cobra.Command, output authcmd.ListClientsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Clients) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Auth clients found.")
		return err
	}
	for _, client := range output.Clients {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", client.ID, client.Name, strings.Join(client.Scopes, ",")); err != nil {
			return err
		}
	}
	return nil
}

func renderAuthClient(cmd *cobra.Command, output authcmd.GetClientOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	client := output.Client
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", client.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", client.Name); err != nil {
		return err
	}
	if len(client.Scopes) > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Scopes\t%s\n", strings.Join(client.Scopes, ",")); err != nil {
			return err
		}
	}
	if len(client.RedirectURIs) > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Redirect URIs\t%s\n", strings.Join(client.RedirectURIs, ",")); err != nil {
			return err
		}
	}
	if len(client.Secrets) > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Secrets\t%d\n", len(client.Secrets)); err != nil {
			return err
		}
	}
	return nil
}

func renderAuthClientMutated(cmd *cobra.Command, output authcmd.ClientMutationOutput, action string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Client %s %s.\n", output.Client.ID, action)
	return err
}

func renderAuthClientDeleted(cmd *cobra.Command, output authcmd.DeleteClientOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Client %s deleted.\n", output.ClientID)
	return err
}

func renderAuthSecretCreated(cmd *cobra.Command, output authcmd.CreateSecretOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.Secret.ID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Secret ID: %s\n", output.Secret.ID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Secret %s created for client %s. Use -o json to retrieve the clear secret.\n", output.Secret.Name, output.ClientID)
	return err
}

func renderAuthSecretDeleted(cmd *cobra.Command, output authcmd.DeleteSecretOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Secret %s deleted for client %s.\n", output.SecretID, output.ClientID)
	return err
}

func renderAuthUsers(cmd *cobra.Command, output authcmd.ListUsersOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Users) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Auth users found.")
		return err
	}
	for _, user := range output.Users {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", user.ID, user.Email, user.Subject); err != nil {
			return err
		}
	}
	return nil
}

func renderAuthUser(cmd *cobra.Command, output authcmd.GetUserOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	user := output.User
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", user.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Email\t%s\n", user.Email); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Subject\t%s\n", user.Subject)
	return err
}
