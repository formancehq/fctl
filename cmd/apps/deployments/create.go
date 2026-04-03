package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	components.Deployment
}

type CreateCtrl struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateCtrl)(nil)

func newCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
	}
}

func NewCreate() *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create a deployment for an app"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithStringFlag("name", "", "Deployment name (required)"),
		fctl.WithStringFlag("stack-id", "", "Stack ID (required)"),
		fctl.WithController(NewCreateCtrl()),
	)
}

func (c *CreateCtrl) GetStore() *CreateStore {
	return c.store
}

func (c *CreateCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}
	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	stackID := fctl.GetString(cmd, "stack-id")
	if stackID == "" {
		return nil, fmt.Errorf("--stack-id is required")
	}

	cmd.SilenceUsage = true

	res, err := apiClient.CreateDeployment(cmd.Context(), id, components.CreateDeploymentRequest{
		Name:    name,
		StackID: stackID,
	})
	if err != nil {
		return nil, err
	}

	c.store.Deployment = res.DeploymentResponse.Data

	return c, nil
}

func (c *CreateCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Printfln("Deployment created successfully")
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData([][]string{
			{"Name", "App ID", "Stack ID", "Workspace ID"},
			{c.store.Deployment.Name, c.store.Deployment.AppID, c.store.Deployment.StackID, c.store.Deployment.WorkspaceID},
		}).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
