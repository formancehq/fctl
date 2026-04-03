package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	components.Deployment
}

type ShowCtrl struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowCtrl)(nil)

func newShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show a deployment"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithStringFlag("name", "", "Deployment name (required)"),
		fctl.WithController(NewShowCtrl()),
	)
}

func (c *ShowCtrl) GetStore() *ShowStore {
	return c.store
}

func (c *ShowCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	res, err := apiClient.ReadDeployment(cmd.Context(), id, name)
	if err != nil {
		return nil, err
	}

	c.store.Deployment = res.DeploymentResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.DefaultSection.Println("Deployment")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Name: %s", c.store.Deployment.Name)},
		{Level: 0, Text: fmt.Sprintf("App ID: %s", c.store.Deployment.AppID)},
		{Level: 0, Text: fmt.Sprintf("Stack ID: %s", c.store.Deployment.StackID)},
		{Level: 0, Text: fmt.Sprintf("Workspace ID: %s", c.store.Deployment.WorkspaceID)},
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
