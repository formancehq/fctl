package workflows

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

type WorkflowsCreateStore struct {
	WorkflowId string `json:"workflowId"`
}
type WorkflowsCreateController struct {
	store *WorkflowsCreateStore
}

var _ fctl.Controller[*WorkflowsCreateStore] = (*WorkflowsCreateController)(nil)

func NewDefaultWorkflowsCreateStore() *WorkflowsCreateStore {
	return &WorkflowsCreateStore{}
}

func NewWorkflowsCreateController() *WorkflowsCreateController {
	return &WorkflowsCreateController{
		store: NewDefaultWorkflowsCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand("create <file>|-",
		fctl.WithShortDescription("Create a workflow"),
		fctl.WithAliases("cr", "c"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*WorkflowsCreateStore](NewWorkflowsCreateController()),
	)
}

func (c *WorkflowsCreateController) GetStore() *WorkflowsCreateStore {
	return c.store
}

func (c *WorkflowsCreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	config := shared.WorkflowConfig{}
	if err := yaml.Unmarshal([]byte(script), &config); err != nil {
		return nil, err
	}

	//nolint:gosimple
	response, err := stackClient.Orchestration.V1.
		CreateWorkflow(cmd.Context(), &shared.WorkflowConfig{
			Name:   config.Name,
			Stages: config.Stages,
		})
	if err != nil {
		return nil, err
	}

	c.store.WorkflowId = response.CreateWorkflowResponse.Data.ID

	return c, nil
}

func (c *WorkflowsCreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Workflow created with ID: %s", c.store.WorkflowId)

	return nil
}
