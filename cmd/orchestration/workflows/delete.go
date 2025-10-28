package workflows

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type WorkflowsDeleteStore struct {
	WorkflowId string `json:"workflowId"`
	Success    bool   `json:"success"`
}
type WorkflowsDeleteController struct {
	store *WorkflowsDeleteStore
}

var _ fctl.Controller[*WorkflowsDeleteStore] = (*WorkflowsDeleteController)(nil)

func NewDefaultWorkflowsDeleteStore() *WorkflowsDeleteStore {
	return &WorkflowsDeleteStore{}
}

func NewWorkflowsDeleteController() *WorkflowsDeleteController {
	return &WorkflowsDeleteController{
		store: NewDefaultWorkflowsDeleteStore(),
	}
}
func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete <workflow-id>",
		fctl.WithAliases("del", "d"),
		fctl.WithShortDescription("Soft delete a workflow"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*WorkflowsDeleteStore](NewWorkflowsDeleteController()),
	)
}

func (c *WorkflowsDeleteController) GetStore() *WorkflowsDeleteStore {
	return c.store
}

func (c *WorkflowsDeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	_, err = stackClient.Orchestration.V1.DeleteWorkflow(
		cmd.Context(),
		operations.DeleteWorkflowRequest{
			FlowID: args[0],
		},
	)

	if err != nil {
		return nil, err
	}

	c.store.WorkflowId = args[0]
	c.store.Success = true

	return c, nil
}

func (c *WorkflowsDeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithShowLineNumber().Printfln("Workflow %s Deleted!", c.store.WorkflowId)
	return nil
}
