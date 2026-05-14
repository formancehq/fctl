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

func newCloudOrganizationsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "organizations",
		Short: "Manage Cloud organizations",
	}
	command.AddCommand(newCloudOrganizationsListCommand())
	command.AddCommand(newCloudOrganizationsShowCommand("show", nil, false))
	command.AddCommand(newCloudOrganizationsShowCommand("describe", nil, true))
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
