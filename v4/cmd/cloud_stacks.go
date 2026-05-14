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
	command.AddCommand(newCloudStacksCreateCommand())
	command.AddCommand(newCloudStacksListCommand())
	command.AddCommand(newCloudStacksShowCommand())
	command.AddCommand(newCloudStacksUpdateCommand())
	command.AddCommand(newCloudStacksDeleteCommand())
	command.AddCommand(newCloudStacksEnableCommand())
	command.AddCommand(newCloudStacksDisableCommand())
	command.AddCommand(newCloudStacksRestoreCommand())
	command.AddCommand(newCloudStacksUpgradeCommand())
	command.AddCommand(newCloudStacksUsersCommand())
	command.AddCommand(newCloudStacksModulesCommand())
	return command
}

func newCloudStacksCreateCommand() *cobra.Command {
	var organizationID string
	var regionID string
	var version string
	var metadata []string

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Cloud stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateStackService{Client: client}.Run(cmd.Context(), cloudcmd.CreateStackInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				Name:           args[0],
				RegionID:       regionID,
				Version:        version,
				Metadata:       parsedMetadata,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackMutated(cmd, output, "created")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&regionID, "region", "", "Cloud region ID")
	command.Flags().StringVar(&version, "version", "", "Stack version")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Stack metadata as key=value")
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

func newCloudStacksUpdateCommand() *cobra.Command {
	var organizationID string
	var name string
	var metadata []string

	command := &cobra.Command{
		Use:   "update <stack-id>",
		Short: "Update a Cloud stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			output, err := cloudcmd.UpdateStackService{Client: client}.Run(cmd.Context(), cloudcmd.UpdateStackInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				Name:           name,
				Metadata:       parsedMetadata,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackMutated(cmd, output, "updated")
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&name, "name", "", "Stack name")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Stack metadata as key=value")
	return command
}

func newCloudStacksDeleteCommand() *cobra.Command {
	var organizationID string
	var force bool
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <stack-id>",
		Short: "Delete a Cloud stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud_stacks delete requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteStackService{Client: client}.Run(cmd.Context(), cloudcmd.DeleteStackInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				Force:          force,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackDeleted(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&force, "force", false, "Force Cloud stack deletion")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud stack deletion")
	return command
}

func newCloudStacksEnableCommand() *cobra.Command {
	return newCloudStacksActionCommand("enable", false)
}

func newCloudStacksDisableCommand() *cobra.Command {
	return newCloudStacksActionCommand("disable", true)
}

func newCloudStacksRestoreCommand() *cobra.Command {
	return newCloudStacksActionCommand("restore", true)
}

func newCloudStacksUpgradeCommand() *cobra.Command {
	var organizationID string
	var version string
	var confirm bool

	command := &cobra.Command{
		Use:   "upgrade <stack-id>",
		Short: "Upgrade a Cloud stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud_stacks upgrade requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.StackActionService{Client: client, Action: "upgrade"}.Run(cmd.Context(), cloudcmd.StackActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				Version:        version,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&version, "version", "", "Target stack version; omit for latest")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud stack upgrade")
	return command
}

func newCloudStacksActionCommand(action string, requiresConfirm bool) *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   action + " <stack-id>",
		Short: fmt.Sprintf("%s a Cloud stack", action),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if requiresConfirm && !confirm {
				return fmt.Errorf("cloud_stacks %s requires --confirm", action)
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.StackActionService{Client: client, Action: action}.Run(cmd.Context(), cloudcmd.StackActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	if requiresConfirm {
		command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud stack "+action)
	}
	return command
}

func newCloudStacksUsersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "users",
		Short: "Manage Cloud stack user access",
	}
	command.AddCommand(newCloudStacksUsersListCommand())
	command.AddCommand(newCloudStacksUsersLinkCommand())
	command.AddCommand(newCloudStacksUsersUnlinkCommand())
	return command
}

func newCloudStacksUsersListCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "list <stack-id>",
		Short: "List Cloud stack users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListStackUsersService{Client: client}.Run(cmd.Context(), cloudcmd.StackIDInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackUsers(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudStacksUsersLinkCommand() *cobra.Command {
	var organizationID string
	var policyID int64

	command := &cobra.Command{
		Use:   "link <stack-id> <user-id>",
		Short: "Link a user to a Cloud stack",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.StackUserAccessService{Client: client, Action: "link"}.Run(cmd.Context(), cloudcmd.StackUserAccessInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				UserID:         args[1],
				PolicyID:       policyID,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackUserAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().Int64Var(&policyID, "policy-id", 0, "Cloud stack policy ID")
	return command
}

func newCloudStacksUsersUnlinkCommand() *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   "unlink <stack-id> <user-id>",
		Short: "Unlink a user from a Cloud stack",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud_stacks users unlink requires --confirm")
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.StackUserAccessService{Client: client, Action: "unlink"}.Run(cmd.Context(), cloudcmd.StackUserAccessInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				UserID:         args[1],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackUserAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud stack user unlink")
	return command
}

func newCloudStacksModulesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "modules",
		Short: "Manage Cloud stack modules",
	}
	command.AddCommand(newCloudStacksModulesListCommand())
	command.AddCommand(newCloudStacksModulesEnableCommand())
	command.AddCommand(newCloudStacksModulesDisableCommand())
	return command
}

func newCloudStacksModulesListCommand() *cobra.Command {
	var organizationID string

	command := &cobra.Command{
		Use:   "list <stack-id>",
		Short: "List Cloud stack modules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListModulesService{Client: client}.Run(cmd.Context(), cloudcmd.StackIDInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackModules(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	return command
}

func newCloudStacksModulesEnableCommand() *cobra.Command {
	return newCloudStacksModulesActionCommand("enable", false)
}

func newCloudStacksModulesDisableCommand() *cobra.Command {
	return newCloudStacksModulesActionCommand("disable", true)
}

func newCloudStacksModulesActionCommand(action string, requiresConfirm bool) *cobra.Command {
	var organizationID string
	var confirm bool

	command := &cobra.Command{
		Use:   action + " <stack-id> <module>",
		Short: fmt.Sprintf("%s a Cloud stack module", action),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if requiresConfirm && !confirm {
				return fmt.Errorf("cloud_stacks modules %s requires --confirm", action)
			}
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ModuleActionService{Client: client, Action: action}.Run(cmd.Context(), cloudcmd.ModuleActionInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
				Name:           args[1],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudStackModuleAction(cmd, output)
		},
	}
	command.Flags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	if requiresConfirm {
		command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud stack module "+action)
	}
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

func renderCloudStackMutated(cmd *cobra.Command, output cloudcmd.StackOutput, action string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s %s.\n", output.Stack.ID, action)
	return err
}

func renderCloudStackDeleted(cmd *cobra.Command, output cloudcmd.DeleteStackOutput) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s deleted.\n", output.StackID)
	return err
}

func renderCloudStackAction(cmd *cobra.Command, output cloudcmd.StackActionOutput) error {
	if output.Action == "upgrade" {
		version := output.Version
		if version == "" {
			version = "latest"
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s upgrade requested to %s.\n", output.StackID, version)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s %s requested.\n", output.StackID, output.Action)
	return err
}

func renderCloudStackUsers(cmd *cobra.Command, output cloudcmd.ListStackUsersOutput) error {
	if len(output.Users) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Cloud stack users found.")
		return err
	}
	for _, user := range output.Users {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%d\n", user.UserID, user.Email, user.StackID, user.PolicyID); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudStackUserAction(cmd *cobra.Command, output cloudcmd.StackUserAccessOutput) error {
	done := "linked"
	if output.Action == "unlink" {
		done = "unlinked"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s user %s %s.\n", output.StackID, output.UserID, done)
	return err
}

func renderCloudStackModules(cmd *cobra.Command, output cloudcmd.ListModulesOutput) error {
	if len(output.Modules) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Cloud stack modules found.")
		return err
	}
	for _, module := range output.Modules {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", module.Name, module.State, module.Status); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudStackModuleAction(cmd *cobra.Command, output cloudcmd.ModuleActionOutput) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloud stack %s module %s %sd.\n", output.StackID, output.Name, output.Action)
	return err
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
