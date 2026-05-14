package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	flowscmd "github.com/formancehq/fctl/v4/internal/commands/flows"
)

func newFlowsCommand(deprecatedAlias bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "flows",
		Short: "Manage flows",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if deprecatedAlias {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command orchestration has been deprecated, use flows")
			}
		},
	}
	if deprecatedAlias {
		command.Use = "orchestration"
		command.Deprecated = "use flows"
	}
	command.AddCommand(newFlowsWorkflowsCommand())
	command.AddCommand(newFlowsInstancesCommand())
	return command
}

func newFlowsInstancesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "instances",
		Short: "Manage workflow instances",
	}
	command.AddCommand(newFlowsInstancesListCommand())
	command.AddCommand(newFlowsInstancesShowCommand("show", nil, false))
	command.AddCommand(newFlowsInstancesShowCommand("inspect", nil, false))
	command.AddCommand(newFlowsInstancesShowCommand("describe", nil, true))
	return command
}

func newFlowsInstancesListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var workflowID string
	var running bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List workflow instances",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var runningPtr *bool
			if cmd.Flags().Changed("running") {
				runningPtr = &running
			}
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.ListInstancesService{
				Handlers: flowscmd.SDKListInstancesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureListInstances,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.ListInstancesInput{
				PageSize:   pageSize,
				Cursor:     cursor,
				WorkflowID: workflowID,
				Running:    runningPtr,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsInstances(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&workflowID, "workflow-id", "", "Filter by workflow ID")
	command.Flags().BoolVar(&running, "running", false, "Filter running instances")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsInstancesShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <instance-id>",
		Aliases: aliases,
		Short:   "Show a workflow instance",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command flows instances describe has been deprecated, use flows instances inspect")
			}
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.GetInstanceService{
				Handlers: flowscmd.SDKGetInstanceHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureGetInstance,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.GetInstanceInput{InstanceID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsInstance(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use flows instances inspect"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsWorkflowsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "workflows",
		Short: "Manage workflows",
	}
	command.AddCommand(newFlowsWorkflowsListCommand())
	command.AddCommand(newFlowsWorkflowsShowCommand())
	command.AddCommand(newFlowsWorkflowsCreateCommand())
	command.AddCommand(newFlowsWorkflowsDeleteCommand())
	command.AddCommand(newFlowsWorkflowsRunCommand())
	return command
}

func newFlowsWorkflowsCreateCommand() *cobra.Command {
	var file string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a workflow",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return fmt.Errorf("flows workflows create requires --confirm")
			}
			if file == "" {
				return fmt.Errorf("flows workflows create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			var workflowRequest shared.CreateWorkflowRequest
			if err := json.Unmarshal(data, &workflowRequest); err != nil {
				return fmt.Errorf("decode workflow request: %w", err)
			}
			output, err := runFlowsCreateWorkflowCommand(cmd, workflowRequest, apiVersion)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsWorkflowCreated(cmd, output)
		},
	}
	command.Flags().StringVar(&file, "file", "", "Workflow JSON file path or - for stdin")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm workflow creation")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsWorkflowsDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <workflow-id>",
		Short: "Delete a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("flows workflows delete requires --confirm")
			}
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.DeleteWorkflowService{
				Handlers: flowscmd.SDKDeleteWorkflowHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureDeleteWorkflow,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.DeleteWorkflowInput{WorkflowID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsWorkflowDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm workflow deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsWorkflowsRunCommand() *cobra.Command {
	var variable []string
	var wait bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "run <workflow-id>",
		Short: "Run a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vars, err := parseMetadataFlags(variable)
			if err != nil {
				return err
			}
			var waitPtr *bool
			if cmd.Flags().Changed("wait") {
				waitPtr = &wait
			}
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.RunWorkflowService{
				Handlers: flowscmd.SDKRunWorkflowHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureRunWorkflow,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.RunWorkflowInput{
				WorkflowID: args[0],
				Vars:       vars,
				Wait:       waitPtr,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsWorkflowRun(cmd, output)
		},
	}
	command.Flags().StringArrayVar(&variable, "variable", nil, "Variable as key=value")
	command.Flags().BoolVar(&wait, "wait", false, "Wait for workflow completion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func runFlowsCreateWorkflowCommand(cmd *cobra.Command, workflowRequest shared.CreateWorkflowRequest, apiVersion string) (flowscmd.CreateWorkflowOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return flowscmd.CreateWorkflowOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return flowscmd.CreateWorkflowOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := flowscmd.CreateWorkflowService{
		Handlers: flowscmd.SDKCreateWorkflowHandlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			request := capabilities.VersionResolutionRequest{
				Product:         flowscmd.ProductOrchestration,
				Feature:         flowscmd.FeatureCreateWorkflow,
				HandlerVersions: handlerVersions,
			}
			if apiVersion != "" {
				request.Policy = capabilities.VersionPolicyPinned
				request.PinnedVersion = capabilities.APIVersion(apiVersion)
			}
			return rt.ResolveAPIVersion(ctx, request)
		},
	}
	return service.Run(cmd.Context(), flowscmd.CreateWorkflowInput{Workflow: workflowRequest})
}

func newFlowsWorkflowsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List workflows",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.ListWorkflowsService{
				Handlers: flowscmd.SDKListWorkflowsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureListWorkflows,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.ListWorkflowsInput{
				PageSize: pageSize,
				Cursor:   cursor,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsWorkflows(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsWorkflowsShowCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "show <workflow-id>",
		Short: "Show a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := flowscmd.GetWorkflowService{
				Handlers: flowscmd.SDKGetWorkflowHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureGetWorkflow,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.GetWorkflowInput{WorkflowID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsWorkflow(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func renderFlowsWorkflows(cmd *cobra.Command, output flowscmd.ListWorkflowsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Workflows) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No workflows found.")
		return err
	}
	for _, workflow := range output.Workflows {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", workflow.ID, workflow.Name, workflow.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderFlowsWorkflow(cmd *cobra.Command, output flowscmd.GetWorkflowOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	workflow := output.Workflow
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", workflow.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", workflow.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", workflow.CreatedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated at\t%s\n", workflow.UpdatedAt.Format(time.RFC3339))
	return err
}

func renderFlowsWorkflowCreated(cmd *cobra.Command, output flowscmd.CreateWorkflowOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow created with ID: %s\n", output.Workflow.ID)
	return err
}

func renderFlowsWorkflowDeleted(cmd *cobra.Command, output flowscmd.DeleteWorkflowOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow %s deleted.\n", output.WorkflowID)
	return err
}

func renderFlowsWorkflowRun(cmd *cobra.Command, output flowscmd.RunWorkflowOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow instance created with ID: %s\n", output.Instance.ID)
	return err
}

func renderFlowsInstances(cmd *cobra.Command, output flowscmd.ListInstancesOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Instances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No workflow instances found.")
		return err
	}
	for _, instance := range output.Instances {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%t\t%s\n", instance.ID, instance.WorkflowID, instance.Terminated, instance.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderFlowsInstance(cmd *cobra.Command, output flowscmd.GetInstanceOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	instance := output.Instance
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", instance.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow ID\t%s\n", instance.WorkflowID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Terminated\t%t\n", instance.Terminated); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", instance.CreatedAt.Format(time.RFC3339))
	return err
}
