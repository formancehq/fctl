package workflows

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

type WorkflowsShowStore struct {
	Workflow shared.Workflow `json:"workflow"`
}
type WorkflowsShowController struct {
	store *WorkflowsShowStore
}

var _ fctl.Controller[*WorkflowsShowStore] = (*WorkflowsShowController)(nil)

func NewDefaultWorkflowsShowStore() *WorkflowsShowStore {
	return &WorkflowsShowStore{}
}

func NewWorkflowsShowController() *WorkflowsShowController {
	return &WorkflowsShowController{
		store: NewDefaultWorkflowsShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <id>",
		fctl.WithShortDescription("Show a workflow"),
		fctl.WithAliases("s"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*WorkflowsShowStore](NewWorkflowsShowController()),
	)
}

func (c *WorkflowsShowController) GetStore() *WorkflowsShowStore {
	return c.store
}

func (c *WorkflowsShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	store := fctl.GetStackStore(cmd.Context())

	response, err := store.Client().Orchestration.V2.
		GetWorkflow(cmd.Context(), operations.V2GetWorkflowRequest{
			FlowID: args[0],
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

func (c *WorkflowsShowController) Render(cmd *cobra.Command, args []string) error {
	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Workflow.ID})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), func() string {
		if c.store.Workflow.Config.Name != nil {
			return *c.store.Workflow.Config.Name
		}
		return ""
	}()})
	tableData = append(tableData, []string{pterm.LightCyan("Created at"), c.store.Workflow.CreatedAt.Format(time.RFC3339)})
	tableData = append(tableData, []string{pterm.LightCyan("Updated at"), c.store.Workflow.UpdatedAt.Format(time.RFC3339)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout())

	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Configuration")
	configAsBytes, err := yaml.Marshal(c.store.Workflow.Config)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(configAsBytes))

	return nil
}
