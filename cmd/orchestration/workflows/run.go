package workflows

import (
	"errors"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/orchestration/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type WorkflowsRunStore struct {
	WorkflowInstance shared.WorkflowInstance `json:"workflowInstance"`
}
type WorkflowsRunController struct {
	store        *WorkflowsRunStore
	variableFlag string
	waitFlag     string
	wait         bool
}

var _ fctl.Controller[*WorkflowsRunStore] = (*WorkflowsRunController)(nil)

func NewDefaultWorkflowsRunStore() *WorkflowsRunStore {
	return &WorkflowsRunStore{}
}

func NewWorkflowsRunController() *WorkflowsRunController {
	return &WorkflowsRunController{
		store:        NewDefaultWorkflowsRunStore(),
		variableFlag: "variable",
		waitFlag:     "wait",
		wait:         false,
	}
}

func NewRunCommand() *cobra.Command {
	c := NewWorkflowsRunController()
	return fctl.NewCommand("run <id>",
		fctl.WithShortDescription("Run a workflow"),
		fctl.WithAliases("r"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithBoolFlag(c.waitFlag, false, "Wait end of the run"),
		fctl.WithStringSliceFlag(c.variableFlag, []string{}, "Variable to pass to the workflow"),
		fctl.WithController[*WorkflowsRunStore](c),
	)
}

func (c *WorkflowsRunController) GetStore() *WorkflowsRunStore {
	return c.store
}

func (c *WorkflowsRunController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	wait := fctl.GetBool(cmd, c.waitFlag)
	variables := make(map[string]string)
	for _, variable := range fctl.GetStringSlice(cmd, c.variableFlag) {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("malformed flag: " + variable)
		}
		variables[parts[0]] = parts[1]
	}

	response, err := stackClient.Orchestration.V1.
		RunWorkflow(cmd.Context(), operations.RunWorkflowRequest{
			RequestBody: variables,
			Wait:        &wait,
			WorkflowID:  args[0],
		})
	if err != nil {
		return nil, err
	}

	c.wait = wait
	c.store.WorkflowInstance = response.RunWorkflowResponse.Data
	return c, nil
}

func (c *WorkflowsRunController) Render(cmd *cobra.Command, args []string) error {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return err
	}
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Workflow instance created with ID: %s", c.store.WorkflowInstance.ID)
	if c.wait {
		w, err := stackClient.Orchestration.V1.GetWorkflow(cmd.Context(), operations.GetWorkflowRequest{
			FlowID: args[0],
		})
		if err != nil {
			panic(err)
		}

		return internal.PrintWorkflowInstance(cmd.OutOrStdout(), w.GetWorkflowResponse.Data, c.store.WorkflowInstance)
	}
	return nil
}
