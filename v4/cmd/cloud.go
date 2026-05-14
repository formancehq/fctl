package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	membership "github.com/formancehq/fctl/internal/membershipclient/v3"
	"github.com/spf13/cobra"

	v4auth "github.com/formancehq/fctl/v4/internal/auth"
	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func newCloudCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cloud",
		Short: "Manage Formance Cloud resources",
	}
	command.AddCommand(newCloudMeCommand())
	command.AddCommand(newCloudOrganizationsCommand())
	command.AddCommand(newCloudRegionsCommand())
	command.AddCommand(newCloudStacksCommand("stacks", "cloud stacks", false))
	command.AddCommand(newCloudAppsCommand())
	command.AddCommand(newUICommand(false))
	return command
}

func newCloudMeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "me",
		Short: "Inspect the connected Cloud user",
	}
	command.AddCommand(newCloudMeShowCommand("show", nil, false))
	command.AddCommand(newCloudMeShowCommand("info", nil, true))
	command.AddCommand(newCloudMeInvitationsCommand())
	return command
}

func newCloudMeShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Show the connected Cloud user",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command cloud me info has been deprecated, use cloud me show")
			}
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.MeService{Client: client}.Run(cmd.Context())
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudMe(cmd, output)
		},
	}
	if deprecated {
		command.Hidden = true
	}
	return command
}

func newCloudMeInvitationsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "invitations",
		Short: "Manage invitations for the connected Cloud user",
	}
	command.AddCommand(newCloudMeInvitationsListCommand())
	command.AddCommand(newCloudMeInvitationsActionCommand("accept"))
	command.AddCommand(newCloudMeInvitationsActionCommand("decline"))
	return command
}

func newCloudMeInvitationsListCommand() *cobra.Command {
	var status string
	var organization string

	command := &cobra.Command{
		Use:   "list",
		Short: "List invitations for the connected Cloud user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListInvitationsService{Client: client}.Run(cmd.Context(), cloudcmd.ListInvitationsInput{
				Status:       status,
				Organization: organization,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudInvitations(cmd, output)
		},
	}
	command.Flags().StringVar(&status, "status", "", "Invitation status")
	command.Flags().StringVar(&organization, "organization", "", "Organization ID")
	return command
}

func newCloudMeInvitationsActionCommand(action string) *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   action + " <invitation-id>",
		Short: action + " a Cloud invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud me invitations %s requires --confirm", action)
			}
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.InvitationActionService{Client: client, Action: action}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudInvitationAction(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm invitation "+action)
	return command
}

func newCloudOrganizationsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "organizations",
		Short: "Manage Cloud organizations",
	}
	command.AddCommand(newCloudOrganizationsCreateCommand())
	command.AddCommand(newCloudOrganizationsListCommand())
	command.AddCommand(newCloudOrganizationsShowCommand("show", nil, false))
	command.AddCommand(newCloudOrganizationsShowCommand("describe", nil, true))
	command.AddCommand(newCloudOrganizationsHistoryCommand())
	command.AddCommand(newCloudOrganizationsUpdateCommand())
	command.AddCommand(newCloudOrganizationsDeleteCommand())
	command.AddCommand(newCloudOrganizationsApplicationsCommand())
	command.AddCommand(newCloudOrganizationsAuthenticationProviderCommand())
	command.AddCommand(newCloudOrganizationsInvitationsCommand())
	command.AddCommand(newCloudOrganizationsOAuthClientsCommand())
	command.AddCommand(newCloudOrganizationsUsersCommand())
	command.AddCommand(newCloudOrganizationsPoliciesCommand())
	return command
}

func newCloudOrganizationsHistoryCommand() *cobra.Command {
	var organizationID string
	var stackID string
	var cursor string
	var pageSize int64
	var action string
	var userID string
	var data string

	command := &cobra.Command{
		Use:     "history [organization-id]",
		Aliases: []string{"hist"},
		Short:   "Query Cloud organization history",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			targetOrganizationID := organizationID
			if len(args) == 1 {
				targetOrganizationID = args[0]
			}
			output, err := cloudcmd.ListLogsService{Client: client}.Run(cmd.Context(), cloudcmd.ListLogsInput{
				OrganizationID: resolveCloudOrganizationID(rt, targetOrganizationID),
				StackID:        stackID,
				Cursor:         cursor,
				PageSize:       pageSize,
				Action:         action,
				UserID:         userID,
				Data:           data,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudLogs(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&stackID, "stack", "", "Cloud stack ID")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().Int64Var(&pageSize, "page-size", 10, "Page size")
	command.Flags().StringVar(&action, "action", "", "Filter by action")
	command.Flags().StringVar(&userID, "user-id", "", "Filter by user ID")
	command.Flags().StringVar(&data, "data", "", "Filter by modified data as key=value")
	return command
}

func newCloudOrganizationsOAuthClientsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "oauth-clients",
		Short: "Manage Cloud organization OAuth clients",
	}
	command.AddCommand(newCloudOrganizationsOAuthClientsCreateCommand())
	command.AddCommand(newCloudOrganizationsOAuthClientsListCommand())
	command.AddCommand(newCloudOrganizationsOAuthClientsShowCommand())
	command.AddCommand(newCloudOrganizationsOAuthClientsUpdateCommand())
	command.AddCommand(newCloudOrganizationsOAuthClientsDeleteCommand())
	return command
}

func newCloudOrganizationsOAuthClientsCreateCommand() *cobra.Command {
	var organizationID string
	var name string
	var description string
	var confirm bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a Cloud organization OAuth client",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations oauth-clients create requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateOAuthClientService{Client: client}.Run(cmd.Context(), cloudcmd.OAuthClientInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Name:           name,
				Description:    description,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOAuthClientCreated(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&name, "name", "", "OAuth client name")
	command.Flags().StringVar(&description, "description", "", "OAuth client description")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm OAuth client creation")
	return command
}

func newCloudOrganizationsOAuthClientsListCommand() *cobra.Command {
	var organizationID string
	var cursor string
	var pageSize int64

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Cloud organization OAuth clients",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListOAuthClientsService{Client: client}.Run(cmd.Context(), cloudcmd.ListOAuthClientsInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Cursor:         cursor,
				PageSize:       pageSize,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOAuthClients(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().Int64Var(&pageSize, "page-size", 0, "Page size")
	return command
}

func newCloudOrganizationsOAuthClientsShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <client-id>",
		Short: "Show a Cloud organization OAuth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadOAuthClientService{Client: client}.Run(cmd.Context(), cloudcmd.OAuthClientInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				ClientID:       args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOAuthClient(cmd, output, false)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsOAuthClientsUpdateCommand() *cobra.Command {
	var organizationID string
	var name string
	var description string
	var confirm bool

	command := &cobra.Command{
		Use:   "update <client-id>",
		Short: "Update a Cloud organization OAuth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations oauth-clients update requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.UpdateOAuthClientService{Client: client}.Run(cmd.Context(), cloudcmd.OAuthClientInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				ClientID:       args[0],
				Name:           name,
				Description:    description,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOAuthClientMutated(cmd, output, "updated")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&name, "name", "", "OAuth client name")
	command.Flags().StringVar(&description, "description", "", "OAuth client description")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm OAuth client update")
	return command
}

func newCloudOrganizationsOAuthClientsDeleteCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <client-id>",
		Short: "Delete a Cloud organization OAuth client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations oauth-clients delete requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteOAuthClientService{Client: client}.Run(cmd.Context(), cloudcmd.OAuthClientInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				ClientID:       args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud OAuth client %s deleted.\n", output.ClientID)
			return err
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm OAuth client deletion")
	return command
}

func newCloudOrganizationsAuthenticationProviderCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "authentication-provider",
		Short: "Manage Cloud organization authentication provider",
	}
	command.AddCommand(newCloudOrganizationsAuthenticationProviderShowCommand())
	command.AddCommand(newCloudOrganizationsAuthenticationProviderConfigureCommand())
	command.AddCommand(newCloudOrganizationsAuthenticationProviderDeleteCommand())
	return command
}

func newCloudOrganizationsAuthenticationProviderShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show",
		Short: "Show Cloud organization authentication provider",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadAuthenticationProviderService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudAuthenticationProvider(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsAuthenticationProviderConfigureCommand() *cobra.Command {
	var organizationID string
	var providerType string
	var name string
	var clientID string
	var clientSecret string
	var clientSecretStdin bool
	var oidcIssuer string
	var oidcDiscovery string
	var microsoftTenant string

	command := &cobra.Command{
		Use:   "configure [type name client-id client-secret]",
		Short: "Configure Cloud organization authentication provider",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || len(args) == 4 {
				return nil
			}
			return fmt.Errorf("accepts either no positional args or the deprecated positional form <type> <name> <client-id> <client-secret>, received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 4 {
				if providerType != "" || name != "" || clientID != "" || clientSecret != "" || clientSecretStdin {
					return fmt.Errorf("deprecated positional authentication provider arguments cannot be combined with --type, --name, --client-id, --client-secret, or --client-secret-stdin")
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional authentication provider arguments have been deprecated, use --type --name --client-id and --client-secret-stdin")
				providerType = args[0]
				name = args[1]
				clientID = args[2]
				clientSecret = args[3]
			}
			if clientSecretStdin {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				clientSecret = strings.TrimSpace(string(data))
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ConfigureAuthenticationProviderService{Client: client}.Run(cmd.Context(), cloudcmd.AuthenticationProviderInput{
				OrganizationID:  resolveCloudOrganizationID(rt, organizationID),
				Type:            providerType,
				Name:            name,
				ClientID:        clientID,
				ClientSecret:    clientSecret,
				OIDCIssuer:      oidcIssuer,
				OIDCDiscovery:   oidcDiscovery,
				MicrosoftTenant: microsoftTenant,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudAuthenticationProviderMutated(cmd, output, "configured")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&providerType, "type", "", "Authentication provider type (github, google, microsoft, oidc)")
	command.Flags().StringVar(&name, "name", "", "Authentication provider name")
	command.Flags().StringVar(&clientID, "client-id", "", "Authentication provider client ID")
	command.Flags().StringVar(&clientSecret, "client-secret", "", "Authentication provider client secret")
	command.Flags().BoolVar(&clientSecretStdin, "client-secret-stdin", false, "Read authentication provider client secret from stdin")
	command.Flags().StringVar(&oidcIssuer, "oidc-issuer", "", "OIDC issuer URL")
	command.Flags().StringVar(&oidcDiscovery, "oidc-discovery-path", "", "OIDC discovery path")
	command.Flags().StringVar(&microsoftTenant, "microsoft-tenant", "", "Microsoft tenant ID")
	return command
}

func newCloudOrganizationsAuthenticationProviderDeleteCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete Cloud organization authentication provider",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations authentication-provider delete requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteAuthenticationProviderService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud authentication provider for organization %s deleted.\n", output.OrganizationID)
			return err
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm authentication provider deletion")
	return command
}

func newCloudOrganizationsApplicationsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "applications",
		Short: "Manage Cloud organization applications",
	}
	command.AddCommand(newCloudOrganizationsApplicationsListCommand())
	command.AddCommand(newCloudOrganizationsApplicationsShowCommand())
	return command
}

func newCloudOrganizationsApplicationsListCommand() *cobra.Command {
	var organizationID string
	var page int64
	var pageSize int64

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Cloud organization applications",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListApplicationsService{Client: client}.Run(cmd.Context(), cloudcmd.ListApplicationsInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Page:           page,
				PageSize:       pageSize,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudApplications(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().Int64Var(&page, "page", 0, "Page number")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	return command
}

func newCloudOrganizationsApplicationsShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <application-id>",
		Short: "Show a Cloud organization application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadApplicationService{Client: client}.Run(cmd.Context(), cloudcmd.ApplicationInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				ApplicationID:  args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudApplication(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudRegionsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "regions",
		Short: "Manage Cloud regions",
	}
	command.AddCommand(newCloudRegionsCreateCommand())
	command.AddCommand(newCloudRegionsListCommand())
	command.AddCommand(newCloudRegionsShowCommand())
	command.AddCommand(newCloudRegionsDeleteCommand())
	return command
}

func newCloudRegionsCreateCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a private Cloud region",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateRegionService{Client: client}.Run(cmd.Context(), cloudcmd.RegionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Name:           args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRegionMutated(cmd, output, "created")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudRegionsListCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Cloud regions",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListRegionsService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRegions(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudRegionsShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <region-id>",
		Short: "Show a Cloud region",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadRegionService{Client: client}.Run(cmd.Context(), cloudcmd.RegionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				RegionID:       args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRegion(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudRegionsDeleteCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <region-id>",
		Short: "Delete a Cloud region",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud regions delete requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteRegionService{Client: client}.Run(cmd.Context(), cloudcmd.RegionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				RegionID:       args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud region %s deleted.\n", output.RegionID)
			return err
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud region deletion")
	return command
}

func newCloudOrganizationsCreateCommand() *cobra.Command {
	var domain string
	var defaultPolicyID int64
	var ownerID string

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Cloud organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateOrganizationService{Client: client}.Run(cmd.Context(), cloudcmd.CreateOrganizationInput{
				Name:            args[0],
				Domain:          domain,
				DefaultPolicyID: optionalInt64(defaultPolicyID),
				OwnerID:         ownerID,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationMutated(cmd, output, "created")
		},
	}
	command.Flags().StringVar(&domain, "domain", "", "Organization domain")
	command.Flags().Int64Var(&defaultPolicyID, "default-policy-id", 0, "Default policy ID")
	command.Flags().StringVar(&ownerID, "owner-id", "", "Organization owner user ID")
	return command
}

func newCloudOrganizationsInvitationsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "invitations",
		Short: "Manage Cloud organization invitations",
	}
	command.AddCommand(newCloudOrganizationsInvitationsListCommand())
	command.AddCommand(newCloudOrganizationsInvitationsSendCommand())
	command.AddCommand(newCloudOrganizationsInvitationsDeleteCommand())
	return command
}

func newCloudOrganizationsUsersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "users",
		Short: "Manage Cloud organization users",
	}
	command.AddCommand(newCloudOrganizationsUsersListCommand())
	command.AddCommand(newCloudOrganizationsUsersShowCommand())
	command.AddCommand(newCloudOrganizationsUsersLinkCommand())
	command.AddCommand(newCloudOrganizationsUsersUnlinkCommand())
	return command
}

func newCloudOrganizationsUsersListCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "list",
		Short: "List Cloud organization users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListOrganizationUsersService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationUsers(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsUsersShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <user-id>",
		Short: "Show a Cloud organization user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadOrganizationUserService{Client: client}.Run(cmd.Context(), cloudcmd.OrganizationUserActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				UserID:         args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationUser(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsUsersLinkCommand() *cobra.Command {
	var organizationID string
	var policyID int64

	command := &cobra.Command{
		Use:   "link <user-id>",
		Short: "Link a user to a Cloud organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.OrganizationUserActionService{Client: client, Action: "link"}.Run(cmd.Context(), cloudcmd.OrganizationUserActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				UserID:         args[0],
				PolicyID:       policyID,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationUserAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().Int64Var(&policyID, "policy-id", 0, "Organization policy ID")
	return command
}

func newCloudOrganizationsUsersUnlinkCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "unlink <user-id>",
		Short: "Unlink a user from a Cloud organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations users unlink requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.OrganizationUserActionService{Client: client, Action: "unlink"}.Run(cmd.Context(), cloudcmd.OrganizationUserActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				UserID:         args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationUserAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm organization user unlink")
	return command
}

func newCloudOrganizationsPoliciesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "policies",
		Short: "Manage Cloud organization policies",
	}
	command.AddCommand(newCloudOrganizationsPoliciesCreateCommand())
	command.AddCommand(newCloudOrganizationsPoliciesListCommand())
	command.AddCommand(newCloudOrganizationsPoliciesShowCommand())
	command.AddCommand(newCloudOrganizationsPoliciesUpdateCommand())
	command.AddCommand(newCloudOrganizationsPoliciesDeleteCommand())
	command.AddCommand(newCloudOrganizationsPoliciesScopeActionCommand("add-scope", false))
	command.AddCommand(newCloudOrganizationsPoliciesScopeActionCommand("remove-scope", true))
	return command
}

func newCloudOrganizationsPoliciesCreateCommand() *cobra.Command {
	var organizationID string
	var description string

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Cloud organization policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreatePolicyService{Client: client}.Run(cmd.Context(), cloudcmd.PolicyInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Name:           args[0],
				Description:    description,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudPolicyMutated(cmd, output, "created")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&description, "description", "", "Policy description")
	return command
}

func newCloudOrganizationsPoliciesListCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "list",
		Short: "List Cloud organization policies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListPoliciesService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudPolicies(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsPoliciesShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <policy-id>",
		Short: "Show a Cloud organization policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			policyID, err := parseInt64Arg("policy id", args[0])
			if err != nil {
				return err
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadPolicyService{Client: client}.Run(cmd.Context(), cloudcmd.PolicyInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				PolicyID:       policyID,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudPolicy(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsPoliciesUpdateCommand() *cobra.Command {
	var organizationID string
	var name string
	var description string

	command := &cobra.Command{
		Use:   "update <policy-id>",
		Short: "Update a Cloud organization policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			policyID, err := parseInt64Arg("policy id", args[0])
			if err != nil {
				return err
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.UpdatePolicyService{Client: client}.Run(cmd.Context(), cloudcmd.PolicyInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				PolicyID:       policyID,
				Name:           name,
				Description:    description,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudPolicyMutated(cmd, output, "updated")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&name, "name", "", "Policy name")
	command.Flags().StringVar(&description, "description", "", "Policy description")
	return command
}

func newCloudOrganizationsPoliciesDeleteCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete a Cloud organization policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations policies delete requires --confirm")
			}
			return runCloudPolicyAction(cmd, organizationID, args[0], 0, "delete")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm policy deletion")
	return command
}

func newCloudOrganizationsPoliciesScopeActionCommand(action string, requiresConfirm bool) *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   action + " <policy-id> <scope-id>",
		Short: action + " on a Cloud organization policy",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if requiresConfirm && !confirm {
				return fmt.Errorf("cloud organizations policies %s requires --confirm", action)
			}
			scopeID, err := parseInt64Arg("scope id", args[1])
			if err != nil {
				return err
			}
			return runCloudPolicyAction(cmd, organizationID, args[0], scopeID, action)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	if requiresConfirm {
		command.Flags().BoolVar(&confirm, "confirm", false, "Confirm policy scope removal")
	}
	return command
}

func runCloudPolicyAction(cmd *cobra.Command, organizationID string, policyIDArg string, scopeID int64, action string) error {
	policyID, err := parseInt64Arg("policy id", policyIDArg)
	if err != nil {
		return err
	}
	rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
	if err != nil {
		return err
	}
	output, err := cloudcmd.PolicyActionService{Client: client, Action: action}.Run(cmd.Context(), cloudcmd.PolicyActionInput{
		OrganizationID: resolveCloudOrganizationID(rt, organizationID),
		PolicyID:       policyID,
		ScopeID:        scopeID,
	})
	if err != nil {
		return err
	}
	if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
		return err
	}
	return renderCloudPolicyAction(cmd, output)
}

func newCloudOrganizationsInvitationsListCommand() *cobra.Command {
	var organizationID string
	var status string

	command := &cobra.Command{
		Use:   "list",
		Short: "List Cloud organization invitations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListOrganizationInvitationsService{Client: client}.Run(cmd.Context(), cloudcmd.ListOrganizationInvitationsInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Status:         status,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudInvitations(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&status, "status", "", "Invitation status")
	return command
}

func newCloudOrganizationsInvitationsSendCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "send <email>",
		Short: "Send a Cloud organization invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			invitation, err := cloudcmd.CreateInvitationService{Client: client}.Run(cmd.Context(), cloudcmd.CreateInvitationInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Email:          args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, map[string]cloudcmd.InvitationSummary{"invitation": invitation}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud invitation %s sent.\n", invitation.ID)
			return err
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudOrganizationsInvitationsDeleteCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <invitation-id>",
		Short: "Delete a Cloud organization invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations invitations delete requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteInvitationService{Client: client}.Run(cmd.Context(), cloudcmd.OrganizationInvitationActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				InvitationID:   args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud invitation %s deleted.\n", output.InvitationID)
			return err
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm invitation deletion")
	return command
}

func newCloudOrganizationsListCommand() *cobra.Command {
	var expand bool

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Cloud organizations",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListOrganizationsService{Client: client}.Run(cmd.Context(), cloudcmd.ListOrganizationsInput{Expand: expand})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizations(cmd, output)
		},
	}
	command.Flags().BoolVar(&expand, "expand", false, "Expand related organization data")
	return command
}

func newCloudOrganizationsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var expand bool

	command := &cobra.Command{
		Use:     use + " <organization-id>",
		Aliases: aliases,
		Short:   "Show a Cloud organization",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command cloud organizations describe has been deprecated, use cloud organizations show")
			}
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ReadOrganizationService{Client: client}.Run(cmd.Context(), cloudcmd.OrganizationIDInput{
				OrganizationID: args[0],
				Expand:         expand,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganization(cmd, output)
		},
	}
	command.Flags().BoolVar(&expand, "expand", false, "Expand related organization data")
	if deprecated {
		command.Hidden = true
	}
	return command
}

func newCloudOrganizationsUpdateCommand() *cobra.Command {
	var name string
	var domain string
	var defaultPolicyID int64

	command := &cobra.Command{
		Use:   "update <organization-id>",
		Short: "Update a Cloud organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.UpdateOrganizationService{Client: client}.Run(cmd.Context(), cloudcmd.UpdateOrganizationInput{
				OrganizationID:  args[0],
				Name:            name,
				Domain:          domain,
				DefaultPolicyID: optionalInt64(defaultPolicyID),
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudOrganizationMutated(cmd, output, "updated")
		},
	}
	command.Flags().StringVar(&name, "name", "", "Organization name")
	command.Flags().StringVar(&domain, "domain", "", "Organization domain")
	command.Flags().Int64Var(&defaultPolicyID, "default-policy-id", 0, "Default policy ID")
	return command
}

func newCloudOrganizationsDeleteCommand() *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <organization-id>",
		Short: "Delete a Cloud organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud organizations delete requires --confirm")
			}
			client, err := membershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteOrganizationService{Client: client}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud organization %s deleted.\n", output.OrganizationID)
			return err
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud organization deletion")
	return command
}

func optionalInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func parseInt64Arg(name string, value string) (int64, error) {
	ret, err := strconv.ParseInt(value, 10, 64)
	if err != nil || ret <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return ret, nil
}

func membershipClientFromCommand(cmd *cobra.Command) (*membership.SDK, error) {
	_, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
	return client, err
}

func cloudRuntimeAndMembershipClientFromCommand(cmd *cobra.Command) (*runtime.Runtime, *membership.SDK, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return nil, nil, err
	}
	if rt.Target.Kind != runtime.TargetKindCloud && rt.Target.Kind != runtime.TargetKindCloudStack {
		return nil, nil, fmt.Errorf("cloud commands require a cloud or cloud-stack context")
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return nil, nil, err
	}
	return rt, newMembershipClient(rt.Target.URL, httpClient), nil
}

func organizationMembershipClientFromRuntime(cmd *cobra.Command, rt *runtime.Runtime, organizationID string) (*membership.SDK, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is required")
	}
	if rt.Context.Auth.Method != v4config.AuthMethodCloudDevice {
		httpClient, err := rt.HTTPClient(cmd.Context())
		if err != nil {
			return nil, err
		}
		return newMembershipClient(rt.Target.URL, httpClient), nil
	}
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	scopedAuth, err := ensureCloudDeviceOrganizationAuth(cmd, rt, organizationID)
	if err != nil {
		return nil, err
	}
	httpClient, err := v4auth.NewHTTPClient(cmd.Context(), scopedAuth, rt.Credentials, rt.AuthOptions)
	if err != nil {
		return nil, err
	}
	return newMembershipClient(rt.Target.URL, httpClient), nil
}

func ensureCloudDeviceOrganizationAuth(cmd *cobra.Command, rt *runtime.Runtime, organizationID string) (v4config.Auth, error) {
	rootAuth := rt.Context.Auth
	scopedRef := organizationTokenRef(rootAuth.TokenRef, organizationID)
	issuerURL := rootAuth.IssuerURL
	if issuerURL == "" {
		issuerURL = rt.Context.CloudURL
	}
	scopedAuth := v4config.Auth{
		Method:    v4config.AuthMethodCloudDevice,
		IssuerURL: issuerURL,
		TokenRef:  scopedRef,
		Scopes:    append([]string(nil), v4auth.OrganizationScopes...),
	}
	source, err := v4auth.NewTokenSource(scopedAuth, rt.Credentials, rt.AuthOptions)
	if err == nil {
		if _, tokenErr := source.Token(cmd.Context()); tokenErr == nil {
			return scopedAuth, nil
		} else if !isCredentialNotFound(tokenErr) {
			return v4config.Auth{}, tokenErr
		}
	}

	rootValue, err := rt.Credentials.Get(cmd.Context(), rootAuth.TokenRef)
	if err != nil {
		return v4config.Auth{}, err
	}
	rootTokens, err := v4auth.ParseDeviceTokens(rootValue)
	if err != nil {
		return v4config.Auth{}, err
	}
	authOptions, err := authOptionsFromCommand(cmd)
	if err != nil {
		return v4config.Auth{}, err
	}
	tokens, err := v4auth.DeviceLogin(cmd.Context(), v4auth.DeviceLoginOptions{
		IssuerURL:      issuerURL,
		ClientID:       v4auth.DeviceClientID,
		Scopes:         append([]string{"openid", "offline_access"}, v4auth.OrganizationScopes...),
		OrganizationID: organizationID,
		IDTokenHint:    rootTokens.IDToken,
		HTTPClient:     authOptions.HTTPClient,
		OpenURL:        loginOpenURL,
		Out:            cmd.OutOrStdout(),
	})
	if err != nil {
		return v4config.Auth{}, err
	}
	encoded, err := v4auth.MarshalDeviceTokens(tokens)
	if err != nil {
		return v4config.Auth{}, err
	}
	if err := rt.Credentials.Set(cmd.Context(), scopedRef, encoded); err != nil {
		return v4config.Auth{}, err
	}
	return scopedAuth, nil
}

func ensureCloudDeviceStackAuth(cmd *cobra.Command, rt *runtime.Runtime, stack cloudcmd.StackSummary) (v4config.Auth, error) {
	if stack.ID == "" {
		return v4config.Auth{}, fmt.Errorf("stack id is required")
	}
	if rt.Target.Organization == "" {
		return v4config.Auth{}, fmt.Errorf("organization id is required")
	}
	rootAuth := rt.Context.Auth
	scopedRef := stackTokenRef(rootAuth.TokenRef, rt.Target.Organization, stack.ID)
	issuerURL := rootAuth.IssuerURL
	if issuerURL == "" {
		issuerURL = rt.Context.CloudURL
	}
	resource := v4auth.StackResource(rt.Target.Organization, stack.ID)
	scopedAuth := v4config.Auth{
		Method:    v4config.AuthMethodCloudDevice,
		IssuerURL: issuerURL,
		TokenRef:  scopedRef,
	}
	source, err := v4auth.NewTokenSource(scopedAuth, rt.Credentials, rt.AuthOptions)
	if err == nil {
		if _, tokenErr := source.Token(cmd.Context()); tokenErr == nil {
			return scopedAuth, nil
		} else if !isCredentialNotFound(tokenErr) {
			return v4config.Auth{}, tokenErr
		}
	}

	rootValue, err := rt.Credentials.Get(cmd.Context(), rootAuth.TokenRef)
	if err != nil {
		return v4config.Auth{}, err
	}
	rootTokens, err := v4auth.ParseDeviceTokens(rootValue)
	if err != nil {
		return v4config.Auth{}, err
	}
	authOptions, err := authOptionsFromCommand(cmd)
	if err != nil {
		return v4config.Auth{}, err
	}
	tokens, err := v4auth.DeviceLogin(cmd.Context(), v4auth.DeviceLoginOptions{
		IssuerURL:      issuerURL,
		ClientID:       v4auth.DeviceClientID,
		Scopes:         []string{"openid", "offline_access"},
		Resources:      []string{resource},
		OrganizationID: rt.Target.Organization,
		IDTokenHint:    rootTokens.IDToken,
		HTTPClient:     authOptions.HTTPClient,
		OpenURL:        loginOpenURL,
		Out:            cmd.OutOrStdout(),
	})
	if err != nil {
		return v4config.Auth{}, err
	}
	encoded, err := v4auth.MarshalDeviceTokens(tokens)
	if err != nil {
		return v4config.Auth{}, err
	}
	if err := rt.Credentials.Set(cmd.Context(), scopedRef, encoded); err != nil {
		return v4config.Auth{}, err
	}
	return scopedAuth, nil
}

func organizationTokenRef(rootRef string, organizationID string) string {
	base := strings.TrimSuffix(rootRef, "/root-tokens")
	if base == rootRef {
		base = strings.TrimSuffix(rootRef, "/token")
	}
	return base + "/organizations/" + organizationID + "/root-tokens"
}

func stackTokenRef(rootRef string, organizationID string, stackID string) string {
	base := strings.TrimSuffix(rootRef, "/root-tokens")
	if base == rootRef {
		base = strings.TrimSuffix(rootRef, "/token")
	}
	return base + "/organizations/" + organizationID + "/stacks/" + stackID + "/root-tokens"
}

func isCredentialNotFound(err error) bool {
	return strings.Contains(err.Error(), credentials.ErrNotFound.Error())
}

func newMembershipClient(baseURL string, httpClient *http.Client) *membership.SDK {
	options := []membership.SDKOption{membership.WithServerURL(baseURL)}
	if httpClient != nil {
		options = append(options, membership.WithClient(httpClient))
	}
	return membership.New(options...)
}

func renderCloudMe(cmd *cobra.Command, output cloudcmd.MeOutput) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nEmail\t%s\nRole\t%s\n", output.User.ID, output.User.Email, output.User.Role)
	return err
}

func renderCloudInvitations(cmd *cobra.Command, output cloudcmd.ListInvitationsOutput) error {
	if len(output.Invitations) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No invitations found.")
		return err
	}
	for _, invitation := range output.Invitations {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", invitation.ID, invitation.OrganizationID, invitation.UserEmail, invitation.Status); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudInvitationAction(cmd *cobra.Command, output cloudcmd.InvitationActionOutput) error {
	done := "accepted"
	if output.Action == "decline" {
		done = "declined"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud invitation %s %s.\n", output.InvitationID, done)
	return err
}

func renderCloudRegions(cmd *cobra.Command, output cloudcmd.ListRegionsOutput) error {
	if len(output.Regions) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No regions found.")
		return err
	}
	for _, region := range output.Regions {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%t\t%t\n", region.ID, region.Name, region.Active, region.Public); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudRegion(cmd *cobra.Command, output cloudcmd.RegionOutput) error {
	region := output.Region
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nName\t%s\nActive\t%t\nPublic\t%t\n", region.ID, region.Name, region.Active, region.Public); err != nil {
		return err
	}
	if region.BaseURL != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "BaseURL\t%s\n", region.BaseURL); err != nil {
			return err
		}
	}
	if region.Version != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Version\t%s\n", region.Version); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudRegionMutated(cmd *cobra.Command, output cloudcmd.RegionOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud region %s %s.\n", output.Region.ID, action)
	return err
}

func renderCloudOrganizations(cmd *cobra.Command, output cloudcmd.ListOrganizationsOutput) error {
	if len(output.Organizations) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No organizations found.")
		return err
	}
	for _, organization := range output.Organizations {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", organization.ID, organization.Name, organization.OwnerID); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudLogs(cmd *cobra.Command, output cloudcmd.ListLogsOutput) error {
	if len(output.Logs) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No logs found.")
		return err
	}
	for _, log := range output.Logs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", log.Seq, log.OrganizationID, log.UserID, log.Action, log.Date.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudApplications(cmd *cobra.Command, output cloudcmd.ListApplicationsOutput) error {
	if len(output.Applications) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No applications found.")
		return err
	}
	for _, application := range output.Applications {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", application.ID, application.Name, application.Alias, application.URL); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudApplication(cmd *cobra.Command, output cloudcmd.ApplicationOutput) error {
	application := output.Application
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nName\t%s\nAlias\t%s\nURL\t%s\n", application.ID, application.Name, application.Alias, application.URL); err != nil {
		return err
	}
	if application.Description != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Description\t%s\n", application.Description); err != nil {
			return err
		}
	}
	for _, scope := range application.Scopes {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Scope\t%d\t%s\n", scope.ID, scope.Label); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudAuthenticationProvider(cmd *cobra.Command, output cloudcmd.AuthenticationProviderOutput) error {
	provider := output.Provider
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Type\t%s\nName\t%s\nClientID\t%s\n", provider.Type, provider.Name, provider.ClientID); err != nil {
		return err
	}
	if provider.RedirectURI != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "RedirectURI\t%s\n", provider.RedirectURI); err != nil {
			return err
		}
	}
	if provider.Issuer != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Issuer\t%s\n", provider.Issuer); err != nil {
			return err
		}
	}
	if provider.Tenant != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Tenant\t%s\n", provider.Tenant); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudAuthenticationProviderMutated(cmd *cobra.Command, output cloudcmd.AuthenticationProviderOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud authentication provider %s %s.\n", output.Provider.Name, action)
	return err
}

func renderCloudOAuthClients(cmd *cobra.Command, output cloudcmd.ListOAuthClientsOutput) error {
	if len(output.Clients) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No OAuth clients found.")
		return err
	}
	for _, client := range output.Clients {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", client.ClientID, client.Name, client.SecretLastDigits, client.Description); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudOAuthClient(cmd *cobra.Command, output cloudcmd.OAuthClientOutput, includeSecret bool) error {
	client := output.Client
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ClientID\t%s\nName\t%s\nSecretLastDigits\t%s\nDescription\t%s\n", client.ClientID, client.Name, client.SecretLastDigits, client.Description); err != nil {
		return err
	}
	if includeSecret && client.Secret != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Secret\t%s\n", client.Secret); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudOAuthClientCreated(cmd *cobra.Command, output cloudcmd.OAuthClientOutput) error {
	return renderCloudOAuthClient(cmd, output, true)
}

func renderCloudOAuthClientMutated(cmd *cobra.Command, output cloudcmd.OAuthClientOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud OAuth client %s %s.\n", output.Client.ClientID, action)
	return err
}

func renderCloudOrganizationUsers(cmd *cobra.Command, output cloudcmd.ListOrganizationUsersOutput) error {
	if len(output.Users) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No organization users found.")
		return err
	}
	for _, user := range output.Users {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\n", user.ID, user.Email, user.PolicyID); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudOrganizationUser(cmd *cobra.Command, output cloudcmd.OrganizationUserOutput) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nEmail\t%s\nPolicy\t%d\n", output.User.ID, output.User.Email, output.User.PolicyID)
	return err
}

func renderCloudOrganizationUserAction(cmd *cobra.Command, output cloudcmd.OrganizationUserActionOutput) error {
	done := "linked"
	if output.Action == "unlink" {
		done = "unlinked"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud organization %s user %s %s.\n", output.OrganizationID, output.UserID, done)
	return err
}

func renderCloudPolicies(cmd *cobra.Command, output cloudcmd.ListPoliciesOutput) error {
	if len(output.Policies) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No policies found.")
		return err
	}
	for _, policy := range output.Policies {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%t\n", policy.ID, policy.Name, policy.Protected); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudPolicy(cmd *cobra.Command, output cloudcmd.PolicyOutput) error {
	policy := output.Policy
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%d\nName\t%s\nProtected\t%t\n", policy.ID, policy.Name, policy.Protected); err != nil {
		return err
	}
	if policy.Description != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Description\t%s\n", policy.Description); err != nil {
			return err
		}
	}
	for _, scope := range policy.Scopes {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Scope\t%d\t%s\n", scope.ID, scope.Label); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudPolicyMutated(cmd *cobra.Command, output cloudcmd.PolicyOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud policy %d %s.\n", output.Policy.ID, action)
	return err
}

func renderCloudPolicyAction(cmd *cobra.Command, output cloudcmd.PolicyActionOutput) error {
	switch output.Action {
	case "delete":
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud policy %d deleted.\n", output.PolicyID)
		return err
	case "add-scope":
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud policy %d scope %d added.\n", output.PolicyID, output.ScopeID)
		return err
	case "remove-scope":
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud policy %d scope %d removed.\n", output.PolicyID, output.ScopeID)
		return err
	default:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud policy %d %s completed.\n", output.PolicyID, output.Action)
		return err
	}
}

func renderCloudOrganization(cmd *cobra.Command, output cloudcmd.OrganizationOutput) error {
	organization := output.Organization
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nName\t%s\nOwner\t%s\n", organization.ID, organization.Name, organization.OwnerID); err != nil {
		return err
	}
	if organization.Domain != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Domain\t%s\n", organization.Domain); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudOrganizationMutated(cmd *cobra.Command, output cloudcmd.OrganizationOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud organization %s %s.\n", output.Organization.ID, action)
	return err
}
