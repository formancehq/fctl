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
	command.AddCommand(newAuthClientsListCommand())
	command.AddCommand(newAuthClientsShowCommand("show", []string{"sh"}, false))
	command.AddCommand(newAuthClientsShowCommand("get", nil, true))
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
