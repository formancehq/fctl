package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	v4prompt "github.com/formancehq/fctl/v4/internal/prompt"
	v4render "github.com/formancehq/fctl/v4/internal/render"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func newCloudStacksCommand(use string, canonical string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: "Manage Formance Cloud stacks",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if deprecated {
				if use == "stack" && cmd.Name() == "proxy" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Command stack proxy has been deprecated, use target proxy")
					return
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Command %s has been deprecated, use %s\n", use, canonical)
			}
		},
	}
	if deprecated {
		command.Deprecated = "use " + canonical
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
	command.AddCommand(newCloudStacksHistoryCommand())
	command.AddCommand(newCloudStacksUsersCommand())
	command.AddCommand(newCloudStacksModulesCommand())
	if use == "stack" {
		command.AddCommand(newStackProxyCommand())
	}
	return command
}

func newStackProxyCommand() *cobra.Command {
	command := newTargetProxyCommand()
	command.Short = "Deprecated alias for target proxy"
	return command
}

func newCloudStacksCreateCommand() *cobra.Command {
	var organizationID string
	var regionID string
	var version string
	var metadata []string
	var noWait bool

	command := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a Cloud stack",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			input, err := newCloudStackCreateInput(cmd)
			if err != nil {
				return err
			}
			organizationID, err := resolveCloudOrganizationIDOrPrompt(cmd, rt, client, organizationID)
			if err != nil {
				return err
			}
			organizationClient, err := organizationMembershipClientFromRuntime(cmd, rt, organizationID)
			if err != nil {
				return err
			}
			stackName, err := input.resolveName(cmd, args)
			if err != nil {
				return err
			}
			regionID, err := input.resolveRegion(cmd, organizationClient, organizationID, regionID)
			if err != nil {
				return err
			}
			version, err := input.resolveVersion(cmd, organizationClient, organizationID, regionID, version)
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			output, err := withTerminalSpinner(cmd, !noWait, "Creating stack and waiting for availability", "Stack is available", func() (cloudcmd.StackOutput, error) {
				return cloudcmd.CreateStackService{Client: organizationClient}.Run(cmd.Context(), cloudcmd.CreateStackInput{
					OrganizationID: organizationID,
					Name:           stackName,
					RegionID:       regionID,
					Version:        version,
					Metadata:       parsedMetadata,
					Wait:           !noWait,
				})
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
	command.Flags().BoolVar(&noWait, "no-wait", false, "Do not wait for stack availability")
	return command
}

type cloudStackCreateInput struct {
	nonInteractive bool
	wizard         v4prompt.Wizard
	reader         *bufio.Reader
	customInput    bool
}

func newCloudStackCreateInput(cmd *cobra.Command) (cloudStackCreateInput, error) {
	nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
	if err != nil {
		return cloudStackCreateInput{}, err
	}
	in := cmd.InOrStdin()
	_, inputIsFile := in.(*os.File)
	return cloudStackCreateInput{
		nonInteractive: nonInteractive,
		wizard:         v4prompt.NewWizard(in, cmd.ErrOrStderr()),
		reader:         bufio.NewReader(in),
		customInput:    !inputIsFile,
	}, nil
}

func (i cloudStackCreateInput) resolveName(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	value, err := i.input(cmd, "Enter a name", "cloud stacks create requires a stack name in non-interactive mode")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("stack name is required")
	}
	i.report(cmd, "Name", value)
	return value, nil
}

func (i cloudStackCreateInput) resolveRegion(cmd *cobra.Command, client cloudcmd.MembershipClient, organizationID string, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if i.nonInteractive {
		return "", fmt.Errorf("cloud stacks create requires --region in non-interactive mode")
	}
	output, err := cloudcmd.ListRegionsService{Client: client}.Run(cmd.Context(), organizationID)
	if err != nil {
		return "", fmt.Errorf("list regions for selection: %w", err)
	}
	if len(output.Regions) == 0 {
		return "", fmt.Errorf("cloud stacks create requires --region and no regions are available")
	}
	choices := make([]v4prompt.Choice, 0, len(output.Regions))
	for _, region := range output.Regions {
		choices = append(choices, v4prompt.Choice{
			Title: cloudStackRegionChoiceTitle(region),
			Value: region.ID,
		})
	}
	value, err := i.selectRequired(cmd, "Please select a region", choices, "region id is required")
	if err != nil {
		return "", err
	}
	i.report(cmd, "Region", value)
	return value, nil
}

func (i cloudStackCreateInput) resolveVersion(cmd *cobra.Command, client cloudcmd.MembershipClient, organizationID string, regionID string, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if i.nonInteractive || (!i.wizard.Available() && !i.customInput) {
		return "", nil
	}
	output, err := cloudcmd.ListRegionVersionsService{Client: client}.Run(cmd.Context(), cloudcmd.RegionInput{
		OrganizationID: organizationID,
		RegionID:       regionID,
	})
	if err != nil {
		return "", fmt.Errorf("list region versions for selection: %w", err)
	}
	if len(output.Versions) == 0 {
		return "", nil
	}
	versions := sortedCloudStackRegionVersions(output.Versions)
	choices := make([]v4prompt.Choice, 0, len(versions))
	for _, version := range versions {
		if strings.TrimSpace(version.Name) == "" {
			continue
		}
		title := version.Name
		if version.Deprecated {
			title += " (deprecated)"
		}
		choices = append(choices, v4prompt.Choice{Title: title, Value: version.Name})
	}
	if len(choices) == 0 {
		return "", nil
	}
	value, err := i.selectOptional(cmd, "Please select a version", choices)
	if err != nil {
		return "", err
	}
	i.report(cmd, "Version", value)
	return value, nil
}

func (i cloudStackCreateInput) input(cmd *cobra.Command, label string, nonInteractiveMessage string) (string, error) {
	if i.nonInteractive {
		return "", errors.New(nonInteractiveMessage)
	}
	if i.wizard.Available() {
		value, err := i.wizard.Input(label, "", false)
		return strings.TrimSpace(value), err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label); err != nil {
		return "", err
	}
	value, err := i.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (i cloudStackCreateInput) selectRequired(cmd *cobra.Command, title string, choices []v4prompt.Choice, emptyMessage string) (string, error) {
	value, err := i.selectValue(cmd, title, choices)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New(emptyMessage)
	}
	return value, nil
}

func (i cloudStackCreateInput) selectOptional(cmd *cobra.Command, title string, choices []v4prompt.Choice) (string, error) {
	return i.selectValue(cmd, title, choices)
}

func (i cloudStackCreateInput) selectValue(cmd *cobra.Command, title string, choices []v4prompt.Choice) (string, error) {
	if len(choices) == 0 {
		return "", nil
	}
	if i.wizard.Available() {
		return i.wizard.Select(title, choices)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), title); err != nil {
		return "", err
	}
	for index, choice := range choices {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", index+1, choice.Title); err != nil {
			return "", err
		}
	}
	answer, err := i.input(cmd, "Choice", "interactive selection is disabled")
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", nil
	}
	if index, err := strconv.Atoi(answer); err == nil {
		if index < 1 || index > len(choices) {
			return "", fmt.Errorf("choice %d is out of range", index)
		}
		return choices[index-1].Value, nil
	}
	for _, choice := range choices {
		if strings.EqualFold(answer, choice.Value) || strings.EqualFold(answer, choice.Title) {
			return choice.Value, nil
		}
	}
	return "", fmt.Errorf("unsupported choice %q", answer)
}

func (i cloudStackCreateInput) report(cmd *cobra.Command, label string, value string) {
	if i.nonInteractive || strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), styledKeyValueLine(cmd, label, value))
}

func cloudStackRegionChoiceTitle(region cloudcmd.RegionSummary) string {
	visibility := "Private"
	if region.Public {
		visibility = "Public"
	}
	name := strings.TrimSpace(region.Name)
	if name == "" {
		name = "<noname>"
	}
	return fmt.Sprintf("%s | %s | %s", region.ID, visibility, name)
}

func sortedCloudStackRegionVersions(versions []cloudcmd.RegionVersionSummary) []cloudcmd.RegionVersionSummary {
	sorted := append([]cloudcmd.RegionVersionSummary(nil), versions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareCloudStackVersionNames(sorted[i].Name, sorted[j].Name) > 0
	})
	return sorted
}

func compareCloudStackVersionNames(left string, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	leftComparable, leftValid := comparableCloudStackVersionName(left)
	rightComparable, rightValid := comparableCloudStackVersionName(right)
	if leftValid && rightValid {
		if comparison := semver.Compare(leftComparable, rightComparable); comparison != 0 {
			return comparison
		}
		return strings.Compare(left, right)
	}
	if leftValid {
		return 1
	}
	if rightValid {
		return -1
	}
	return strings.Compare(left, right)
}

func comparableCloudStackVersionName(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if semver.IsValid(value) {
		return value, true
	}
	if !strings.HasPrefix(value, "v") {
		return "", false
	}

	core, suffix, _ := strings.Cut(value, "-")
	if suffix != "" {
		suffix = "-" + suffix
	} else if before, after, ok := strings.Cut(value, "+"); ok {
		core = before
		suffix = "+" + after
	}

	parts := strings.Split(strings.TrimPrefix(core, "v"), ".")
	switch len(parts) {
	case 1:
		value = "v" + parts[0] + ".0.0" + suffix
	case 2:
		value = "v" + parts[0] + "." + parts[1] + ".0" + suffix
	default:
		return "", false
	}
	if semver.IsValid(value) {
		return value, true
	}
	return "", false
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
			organizationID, err := resolveCloudOrganizationIDOrPrompt(cmd, rt, client, organizationID)
			if err != nil {
				return err
			}
			organizationClient, err := organizationMembershipClientFromRuntime(cmd, rt, organizationID)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListStacksService{Client: organizationClient}.Run(cmd.Context(), cloudcmd.ListStacksInput{
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
				return fmt.Errorf("cloud stacks delete requires --confirm")
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
				return fmt.Errorf("cloud stacks upgrade requires --confirm")
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
				return fmt.Errorf("cloud stacks %s requires --confirm", action)
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

func newCloudStacksHistoryCommand() *cobra.Command {
	var organizationID string
	var cursor string
	var pageSize int64
	var action string
	var userID string
	var data string

	command := &cobra.Command{
		Use:     "history <stack-id>",
		Aliases: []string{"hist"},
		Short:   "Query Cloud stack history",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListLogsService{Client: client}.Run(cmd.Context(), cloudcmd.ListLogsInput{
				OrganizationID: resolveCloudOrganizationID(rt, organizationID),
				StackID:        args[0],
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
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().Int64Var(&pageSize, "page-size", 10, "Page size")
	command.Flags().StringVar(&action, "action", "", "Filter by action")
	command.Flags().StringVar(&userID, "user-id", "", "Filter by user ID")
	command.Flags().StringVar(&data, "data", "", "Filter by modified data as key=value")
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
				return fmt.Errorf("cloud stacks users unlink requires --confirm")
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
				return fmt.Errorf("cloud stacks modules %s requires --confirm", action)
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

func resolveCloudOrganizationIDOrPrompt(cmd *cobra.Command, rt *runtime.Runtime, client cloudcmd.MembershipClient, explicit string) (string, error) {
	organizationID := resolveCloudOrganizationID(rt, explicit)
	if organizationID != "" {
		return organizationID, nil
	}

	message := "organization id is required; pass --organization or select a cloud-stack profile"
	nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
	if err != nil {
		return "", err
	}
	wizard := v4prompt.NewWizard(cmd.InOrStdin(), cmd.ErrOrStderr())
	if nonInteractive || !wizard.Available() {
		return "", errors.New(message)
	}

	output, err := cloudcmd.ListOrganizationsService{Client: client}.Run(cmd.Context(), cloudcmd.ListOrganizationsInput{})
	if err != nil {
		return "", fmt.Errorf("list organizations for selection: %w", err)
	}
	if len(output.Organizations) == 0 {
		return "", fmt.Errorf("organization id is required and no organizations are available")
	}

	choices := make([]v4prompt.Choice, 0, len(output.Organizations))
	for _, organization := range output.Organizations {
		title := organization.ID
		if strings.TrimSpace(organization.Name) != "" {
			title = fmt.Sprintf("%s (%s)", organization.Name, organization.ID)
		}
		choices = append(choices, v4prompt.Choice{Title: title, Value: organization.ID})
	}
	return wizard.Select("Organization ID is required. Select an organization:", choices)
}

func renderCloudStacks(cmd *cobra.Command, output cloudcmd.ListStacksOutput) error {
	if len(output.Stacks) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No Cloud stacks found.")
		return err
	}
	headers := []string{"ID", "Name", "Dashboard", "Region", "Status", "Audit Enabled"}
	rows := make([][]string, 0, len(output.Stacks))
	for _, stack := range output.Stacks {
		rows = append(rows, []string{
			stack.ID,
			stack.Name,
			stack.Dashboard,
			stack.RegionID,
			stack.State,
			boolPointerLabel(stack.AuditEnabled),
		})
	}
	return v4render.Table(cmd.OutOrStdout(), headers, rows)
}

func boolPointerLabel(value *bool) string {
	if value == nil {
		return "No"
	}
	if *value {
		return "Yes"
	}
	return "No"
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
