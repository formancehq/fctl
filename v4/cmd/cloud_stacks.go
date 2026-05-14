package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func newCloudStacksCommand(use string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: "Manage Formance Cloud stacks",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if deprecated {
				fmt.Fprintf(cmd.ErrOrStderr(), "Command %s has been deprecated, use cloud_stacks\n", use)
			}
		},
	}
	if deprecated {
		command.Deprecated = "use cloud_stacks"
	}
	command.AddCommand(newCloudStacksListCommand())
	command.AddCommand(newCloudStacksShowCommand())
	return command
}

func newCloudStacksListCommand() *cobra.Command {
	var organizationID string
	var all bool

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Cloud stacks",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			organizationID := resolveCloudOrganizationID(rt, organizationID)
			output, err := cloudcmd.ListStacksService{Client: client}.Run(cmd.Context(), cloudcmd.ListStacksInput{
				OrganizationID: organizationID,
				All:            all,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStacks(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&all, "all", false, "Include deleted and disabled stacks")
	return command
}

func newCloudStacksShowCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "show <stack-id>",
		Short: "Show a Cloud stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			organizationID := resolveCloudOrganizationID(rt, organizationID)
			output, err := cloudcmd.ReadStackService{Client: client}.Run(cmd.Context(), cloudcmd.StackIDInput{
				OrganizationID: organizationID,
				StackID:        args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStack(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func resolveCloudOrganizationID(rt *runtime.Runtime, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if rt == nil {
		return ""
	}
	return rt.Target.Organization
}

func renderCloudStacks(cmd *cobra.Command, output cloudcmd.ListStacksOutput) error {
	if len(output.Stacks) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Cloud stacks found.")
		return err
	}
	for _, stack := range output.Stacks {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", stack.ID, stack.Name, stack.Status, stack.URI); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudStack(cmd *cobra.Command, output cloudcmd.StackOutput) error {
	stack := output.Stack
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nName\t%s\nStatus\t%s\nState\t%s\n", stack.ID, stack.Name, stack.Status, stack.State); err != nil {
		return err
	}
	if stack.URI != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "URI\t%s\n", stack.URI); err != nil {
			return err
		}
	}
	if stack.Version != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Version\t%s\n", stack.Version); err != nil {
			return err
		}
	}
	return nil
}
