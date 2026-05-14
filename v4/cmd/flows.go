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
	command.AddCommand(newFlowsTriggersCommand())
	return command
}

func newFlowsTriggersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "triggers",
		Short: "Manage workflow triggers",
	}
	command.AddCommand(newFlowsTriggersCreateCommand())
	command.AddCommand(newFlowsTriggersListCommand())
	command.AddCommand(newFlowsTriggersShowCommand())
	command.AddCommand(newFlowsTriggersDeleteCommand())
	command.AddCommand(newFlowsTriggersTestCommand())
	return command
}

func newFlowsTriggersCreateCommand() *cobra.Command {
	var confirm bool
	var name string
	var filter string
	var version string
	var variable []string
	var apiVersion string

	command := &cobra.Command{
		Use:   "create <event> <workflow-id>",
		Short: "Create a workflow trigger",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("flows triggers create requires --confirm")
			}
			vars, err := parseAnyMapFlags(variable)
			if err != nil {
				return err
			}
			output, err := runFlowsCreateTriggerCommand(cmd, flowscmd.CreateTriggerInput{
				Event:      args[0],
				WorkflowID: args[1],
				Name:       name,
				Filter:     filter,
				Version:    version,
				Vars:       vars,
			}, apiVersion)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsTriggerCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm trigger creation")
	command.Flags().StringVar(&name, "name", "", "Trigger name")
	command.Flags().StringVar(&filter, "filter", "", "Trigger filter expression")
	command.Flags().StringVar(&version, "version", "", "Workflow version")
	command.Flags().StringArrayVar(&variable, "variable", nil, "Variable as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsTriggersListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var name string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List workflow triggers",
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
			service := flowscmd.ListTriggersService{
				Handlers: flowscmd.SDKListTriggersHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureListTriggers,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.ListTriggersInput{PageSize: pageSize, Cursor: cursor, Name: name})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsTriggers(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&name, "name", "", "Filter triggers by name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsTriggersShowCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "show <trigger-id>",
		Short: "Show a workflow trigger",
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
			service := flowscmd.GetTriggerService{
				Handlers: flowscmd.SDKGetTriggerHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         flowscmd.ProductOrchestration,
						Feature:         flowscmd.FeatureReadTrigger,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), flowscmd.GetTriggerInput{TriggerID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsTrigger(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsTriggersDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <trigger-id>",
		Short: "Delete a workflow trigger",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("flows triggers delete requires --confirm")
			}
			output, err := runFlowsDeleteTriggerCommand(cmd, args[0], apiVersion)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsTriggerDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm trigger deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsTriggersTestCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "test <trigger-id> <event-json>",
		Short: "Test a workflow trigger",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			event := map[string]any{}
			if err := json.Unmarshal([]byte(args[1]), &event); err != nil {
				return fmt.Errorf("decode event json: %w", err)
			}
			output, err := runFlowsTestTriggerCommand(cmd, flowscmd.TestTriggerInput{TriggerID: args[0], Event: event}, apiVersion)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsTriggerTest(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
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
	command.AddCommand(newFlowsInstancesSendEventCommand())
	command.AddCommand(newFlowsInstancesStopCommand())
	return command
}

func newFlowsInstancesSendEventCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "send-event <instance-id> <event>",
		Short: "Send an event to a workflow instance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := runFlowsInstanceActionCommand(cmd, flowsInstanceActionCommandRequest{
				Feature:    flowscmd.FeatureSendEvent,
				Handlers:   flowscmd.SDKSendEventHandlers,
				InstanceID: args[0],
				Event:      args[1],
				APIVersion: apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsInstanceEventSent(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

func newFlowsInstancesStopCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "stop <instance-id>",
		Short: "Stop a workflow instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("flows instances stop requires --confirm")
			}
			output, err := runFlowsInstanceActionCommand(cmd, flowsInstanceActionCommandRequest{
				Feature:    flowscmd.FeatureCancelEvent,
				Handlers:   flowscmd.SDKStopInstanceHandlers,
				InstanceID: args[0],
				APIVersion: apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderFlowsInstanceStopped(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm instance stop")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin orchestration API version")
	return command
}

type flowsInstanceActionCommandRequest struct {
	Feature    capabilities.Feature
	Handlers   func(*formance.Formance) []flowscmd.InstanceActionHandler
	InstanceID string
	Event      string
	APIVersion string
}

func runFlowsInstanceActionCommand(cmd *cobra.Command, request flowsInstanceActionCommandRequest) (flowscmd.InstanceActionOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return flowscmd.InstanceActionOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return flowscmd.InstanceActionOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := flowscmd.StopInstanceService{
		Handlers: request.Handlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			versionRequest := capabilities.VersionResolutionRequest{
				Product:         flowscmd.ProductOrchestration,
				Feature:         request.Feature,
				HandlerVersions: handlerVersions,
			}
			if request.APIVersion != "" {
				versionRequest.Policy = capabilities.VersionPolicyPinned
				versionRequest.PinnedVersion = capabilities.APIVersion(request.APIVersion)
			}
			return rt.ResolveAPIVersion(ctx, versionRequest)
		},
	}
	return service.Run(cmd.Context(), flowscmd.InstanceActionInput{InstanceID: request.InstanceID, Event: request.Event})
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

func runFlowsCreateTriggerCommand(cmd *cobra.Command, input flowscmd.CreateTriggerInput, apiVersion string) (flowscmd.CreateTriggerOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return flowscmd.CreateTriggerOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return flowscmd.CreateTriggerOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := flowscmd.CreateTriggerService{
		Handlers: flowscmd.SDKCreateTriggerHandlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			request := capabilities.VersionResolutionRequest{
				Product:         flowscmd.ProductOrchestration,
				Feature:         flowscmd.FeatureCreateTrigger,
				HandlerVersions: handlerVersions,
			}
			if apiVersion != "" {
				request.Policy = capabilities.VersionPolicyPinned
				request.PinnedVersion = capabilities.APIVersion(apiVersion)
			}
			return rt.ResolveAPIVersion(ctx, request)
		},
	}
	return service.Run(cmd.Context(), input)
}

func runFlowsDeleteTriggerCommand(cmd *cobra.Command, triggerID string, apiVersion string) (flowscmd.DeleteTriggerOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return flowscmd.DeleteTriggerOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return flowscmd.DeleteTriggerOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := flowscmd.DeleteTriggerService{
		Handlers: flowscmd.SDKDeleteTriggerHandlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			request := capabilities.VersionResolutionRequest{
				Product:         flowscmd.ProductOrchestration,
				Feature:         flowscmd.FeatureDeleteTrigger,
				HandlerVersions: handlerVersions,
			}
			if apiVersion != "" {
				request.Policy = capabilities.VersionPolicyPinned
				request.PinnedVersion = capabilities.APIVersion(apiVersion)
			}
			return rt.ResolveAPIVersion(ctx, request)
		},
	}
	return service.Run(cmd.Context(), flowscmd.DeleteTriggerInput{TriggerID: triggerID})
}

func runFlowsTestTriggerCommand(cmd *cobra.Command, input flowscmd.TestTriggerInput, apiVersion string) (flowscmd.TestTriggerOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return flowscmd.TestTriggerOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return flowscmd.TestTriggerOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := flowscmd.TestTriggerService{
		Handlers: flowscmd.SDKTestTriggerHandlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			request := capabilities.VersionResolutionRequest{
				Product:         flowscmd.ProductOrchestration,
				Feature:         flowscmd.FeatureTestTrigger,
				HandlerVersions: handlerVersions,
			}
			if apiVersion != "" {
				request.Policy = capabilities.VersionPolicyPinned
				request.PinnedVersion = capabilities.APIVersion(apiVersion)
			}
			return rt.ResolveAPIVersion(ctx, request)
		},
	}
	return service.Run(cmd.Context(), input)
}

func parseAnyMapFlags(values []string) (map[string]any, error) {
	parsed, err := parseMetadataFlags(values)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, nil
	}
	ret := make(map[string]any, len(parsed))
	for key, value := range parsed {
		ret[key] = value
	}
	return ret, nil
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

func renderFlowsInstanceEventSent(cmd *cobra.Command, output flowscmd.InstanceActionOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Event %s sent to instance %s.\n", output.Event, output.InstanceID)
	return err
}

func renderFlowsInstanceStopped(cmd *cobra.Command, output flowscmd.InstanceActionOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow instance %s stopped.\n", output.InstanceID)
	return err
}

func renderFlowsTriggerCreated(cmd *cobra.Command, output flowscmd.CreateTriggerOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Trigger created with ID: %s\n", output.Trigger.ID)
	return err
}

func renderFlowsTriggers(cmd *cobra.Command, output flowscmd.ListTriggersOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Triggers) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No triggers found.")
		return err
	}
	for _, trigger := range output.Triggers {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", trigger.ID, trigger.Name, trigger.Event, trigger.WorkflowID); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderFlowsTrigger(cmd *cobra.Command, output flowscmd.GetTriggerOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	trigger := output.Trigger
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", trigger.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", trigger.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Event\t%s\n", trigger.Event); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workflow ID\t%s\n", trigger.WorkflowID)
	return err
}

func renderFlowsTriggerDeleted(cmd *cobra.Command, output flowscmd.DeleteTriggerOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Trigger %s deleted.\n", output.TriggerID)
	return err
}

func renderFlowsTriggerTest(cmd *cobra.Command, output flowscmd.TestTriggerOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.Matched != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Filter match\t%t\n", *output.Matched)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Trigger %s tested.\n", output.TriggerID)
	return err
}
