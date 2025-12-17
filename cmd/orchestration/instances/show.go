package instances

import (
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/orchestration/internal"
	fctl "github.com/formancehq/fctl/pkg"
)

type InstancesShowStore struct {
	WorkflowInstance shared.V2WorkflowInstance `json:"workflowInstance"`
	Workflow         shared.Workflow           `json:"workflow"`
}
type InstancesShowController struct {
	store *InstancesShowStore
}

var _ fctl.Controller[*InstancesShowStore] = (*InstancesShowController)(nil)

func NewDefaultInstancesShowStore() *InstancesShowStore {
	return &InstancesShowStore{}
}

func NewInstancesShowController() *InstancesShowController {
	return &InstancesShowController{
		store: NewDefaultInstancesShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <instance-id>",
		fctl.WithShortDescription("Show a specific workflow instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*InstancesShowStore](NewInstancesShowController()),
	)
}

func (c *InstancesShowController) GetStore() *InstancesShowStore {
	return c.store
}

func (c *InstancesShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	res, err := store.Client().Orchestration.V2.GetInstance(cmd.Context(), operations.V2GetInstanceRequest{
		InstanceID: args[0],
	})
	if err != nil {
		return nil, errors.Wrap(err, "reading instance")
	}

	c.store.WorkflowInstance = res.V2GetWorkflowInstanceResponse.Data
	response, err := store.Client().Orchestration.V2.GetWorkflow(cmd.Context(), operations.V2GetWorkflowRequest{
		FlowID: res.V2GetWorkflowInstanceResponse.Data.WorkflowID,
	})
	if err != nil {
		return nil, err
	}

	// Convert V2Workflow to Workflow
	v2Workflow := response.V2GetWorkflowResponse.Data
	c.store.Workflow = shared.Workflow{
		ID:        v2Workflow.ID,
		CreatedAt: v2Workflow.CreatedAt,
		UpdatedAt: v2Workflow.UpdatedAt,
		Config:    shared.WorkflowConfig(v2Workflow.Config),
	}

	return c, nil
}

func (c *InstancesShowController) Render(cmd *cobra.Command, args []string) error {
	// Print the instance information
	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.WorkflowInstance.ID})
	tableData = append(tableData, []string{pterm.LightCyan("Created at"), c.store.WorkflowInstance.CreatedAt.Format(time.RFC3339)})
	tableData = append(tableData, []string{pterm.LightCyan("Updated at"), c.store.WorkflowInstance.UpdatedAt.Format(time.RFC3339)})
	if c.store.WorkflowInstance.Terminated {
		tableData = append(tableData, []string{pterm.LightCyan("Terminated at"), c.store.WorkflowInstance.TerminatedAt.Format(time.RFC3339)})
	}
	if c.store.WorkflowInstance.Error != nil && *c.store.WorkflowInstance.Error != "" {
		tableData = append(tableData, []string{pterm.LightCyan("Error"), *c.store.WorkflowInstance.Error})
	}

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	// Convert V2WorkflowInstance to WorkflowInstance
	v2Instance := c.store.WorkflowInstance
	instance := shared.WorkflowInstance{
		ID:           v2Instance.ID,
		WorkflowID:   v2Instance.WorkflowID,
		CreatedAt:    v2Instance.CreatedAt,
		UpdatedAt:    v2Instance.UpdatedAt,
		Terminated:   v2Instance.Terminated,
		TerminatedAt: v2Instance.TerminatedAt,
		Error:        v2Instance.Error,
		Status: fctl.Map(v2Instance.Status, func(src shared.V2StageStatus) shared.StageStatus {
			return shared.StageStatus{
				StartedAt:    src.StartedAt,
				TerminatedAt: src.TerminatedAt,
				Error:        src.Error,
			}
		}),
	}
	if err := internal.PrintWorkflowInstance(cmd.OutOrStdout(), c.store.Workflow, instance); err != nil {
		return err
	}

	return nil
}
