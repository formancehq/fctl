package cmd

import (
	"fmt"
	"net/http"

	membership "github.com/formancehq/fctl/internal/membershipclient/v3"
	"github.com/spf13/cobra"

	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func newCloudCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cloud",
		Short: "Manage Formance Cloud resources",
	}
	command.AddCommand(newCloudMeCommand())
	command.AddCommand(newCloudOrganizationsCommand())
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
	command.AddCommand(newCloudOrganizationsUpdateCommand())
	command.AddCommand(newCloudOrganizationsDeleteCommand())
	command.AddCommand(newCloudOrganizationsInvitationsCommand())
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
