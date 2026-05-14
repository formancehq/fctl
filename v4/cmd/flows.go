package cmd

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
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
	return command
}

func newFlowsWorkflowsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "workflows",
		Short: "Manage workflows",
	}
	command.AddCommand(newFlowsWorkflowsListCommand())
	command.AddCommand(newFlowsWorkflowsShowCommand())
	return command
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
